package scraper

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// timedText represents the YouTube timedtext XML response.
type timedText struct {
	XMLName xml.Name    `xml:"transcript"`
	Texts   []textEntry `xml:"text"`
}

type textEntry struct {
	Start string `xml:"start,attr"`
	Dur   string `xml:"dur,attr"`
	Text  string `xml:",chardata"`
}

var bracketNoise = regexp.MustCompile(`\[(?:Music|Applause|Laughter|Cheering|Inaudible)\]`)
var multiSpace = regexp.MustCompile(`\s+`)

// GetTranscript fetches the transcript for a YouTube video.
func GetTranscript(ctx context.Context, client *http.Client, videoID string) fn.Result[string] {
	// Try the timedtext XML endpoint (works for many videos without API key)
	urls := []string{
		fmt.Sprintf("https://www.youtube.com/api/timedtext?v=%s&lang=en&fmt=srv3", videoID),
		fmt.Sprintf("https://www.youtube.com/api/timedtext?v=%s&lang=en&kind=asr&fmt=srv3", videoID),
	}

	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || len(body) < 50 {
			continue
		}

		var tt timedText
		if err := xml.Unmarshal(body, &tt); err != nil || len(tt.Texts) == 0 {
			continue
		}

		var sb strings.Builder
		for _, t := range tt.Texts {
			sb.WriteString(t.Text)
			sb.WriteByte(' ')
		}
		return fn.Ok(CleanTranscript(sb.String()))
	}

	return fn.Err[string](fmt.Errorf("no transcript available for video %s", videoID))
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
