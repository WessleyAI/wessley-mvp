package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// timedText represents the YouTube timedtext XML response (srv3 format).
type timedText struct {
	XMLName xml.Name   `xml:"timedtext"`
	Body    ttBody     `xml:"body"`
}

type ttBody struct {
	Paragraphs []ttParagraph `xml:"p"`
}

type ttParagraph struct {
	Start int    `xml:"t,attr"`
	Dur   int    `xml:"d,attr"`
	Text  string `xml:",chardata"`
}

// legacyTimedText is the older transcript XML format.
type legacyTimedText struct {
	XMLName xml.Name       `xml:"transcript"`
	Texts   []legacyEntry  `xml:"text"`
}

type legacyEntry struct {
	Start string `xml:"start,attr"`
	Dur   string `xml:"dur,attr"`
	Text  string `xml:",chardata"`
}

var bracketNoise = regexp.MustCompile(`\[(?:Music|Applause|Laughter|Cheering|Inaudible)\]`)
var multiSpace = regexp.MustCompile(`\s+`)

// captionTrack from the innertube player response.
type captionTrack struct {
	BaseURL string `json:"baseUrl"`
	Lang    string `json:"languageCode"`
	Kind    string `json:"kind"`
}

// GetTranscript fetches the transcript for a YouTube video using the innertube API.
func GetTranscript(ctx context.Context, client *http.Client, videoID string) fn.Result[string] {
	tracks, err := fetchCaptionTracks(ctx, client, videoID)
	if err != nil {
		return fn.Err[string](fmt.Errorf("no transcript available for video %s: %w", videoID, err))
	}

	// Prioritize: English manual captions > English ASR > any language
	var urls []string
	for _, t := range tracks {
		if t.Lang == "en" && t.Kind != "asr" {
			urls = append([]string{t.BaseURL + "&fmt=srv3"}, urls...)
		} else if t.Lang == "en" {
			urls = append(urls, t.BaseURL+"&fmt=srv3")
		}
	}
	if len(urls) == 0 {
		for _, t := range tracks {
			urls = append(urls, t.BaseURL+"&fmt=srv3")
		}
	}

	for _, u := range urls {
		text, err := fetchTranscriptFromURL(ctx, client, u)
		if err == nil && text != "" {
			return fn.Ok(text)
		}
	}

	return fn.Err[string](fmt.Errorf("no transcript available for video %s", videoID))
}

// fetchCaptionTracks uses the YouTube innertube API (ANDROID client) to get caption track URLs.
func fetchCaptionTracks(ctx context.Context, client *http.Client, videoID string) ([]captionTrack, error) {
	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":        "ANDROID",
				"clientVersion":     "19.09.37",
				"androidSdkVersion": 30,
				"hl":                "en",
				"gl":                "US",
			},
		},
		"videoId":        videoID,
		"contentCheckOk": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.youtube.com/youtubei/v1/player?key=AIzaSyA8eiZmM1FaDVjRy-df2KTyQ_vz_yYM39w&prettyPrint=false",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "com.google.android.youtube/19.09.37 (Linux; U; Android 11) gzip")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the nested JSON response to extract caption tracks
	var result struct {
		Captions struct {
			PlayerCaptionsTracklistRenderer struct {
				CaptionTracks []captionTrack `json:"captionTracks"`
			} `json:"playerCaptionsTracklistRenderer"`
		} `json:"captions"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode player response: %w", err)
	}

	tracks := result.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
	if len(tracks) == 0 {
		return nil, fmt.Errorf("no caption tracks in player response")
	}

	return tracks, nil
}

func fetchTranscriptFromURL(ctx context.Context, client *http.Client, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "com.google.android.youtube/19.09.37 (Linux; U; Android 11) gzip")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 || len(body) < 50 {
		return "", fmt.Errorf("bad response: status=%d len=%d", resp.StatusCode, len(body))
	}

	// Try srv3 format first (newer: <timedtext><body><p t="" d="">...)
	var tt timedText
	if err := xml.Unmarshal(body, &tt); err == nil && len(tt.Body.Paragraphs) > 0 {
		var sb strings.Builder
		for _, p := range tt.Body.Paragraphs {
			sb.WriteString(p.Text)
			sb.WriteByte(' ')
		}
		return CleanTranscript(sb.String()), nil
	}

	// Try legacy format (<transcript><text start="" dur="">...)
	var legacy legacyTimedText
	if err := xml.Unmarshal(body, &legacy); err == nil && len(legacy.Texts) > 0 {
		var sb strings.Builder
		for _, t := range legacy.Texts {
			sb.WriteString(t.Text)
			sb.WriteByte(' ')
		}
		return CleanTranscript(sb.String()), nil
	}

	return "", fmt.Errorf("no text entries in transcript")
}

// CleanTranscript removes bracket noise, collapses whitespace, and trims.
func CleanTranscript(text string) string {
	text = bracketNoise.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = multiSpace.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
