package manuals

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// NHTSASource discovers technical documents from NHTSA APIs.
type NHTSASource struct {
	client *http.Client
}

func NewNHTSASource() *NHTSASource {
	return &NHTSASource{client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *NHTSASource) Name() string { return "nhtsa" }

func (s *NHTSASource) Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error) {
	var entries []graph.ManualEntry

	for _, make_ := range makes {
		for _, year := range years {
			select {
			case <-ctx.Done():
				return entries, ctx.Err()
			default:
			}

			found, err := s.fetchRecalls(ctx, make_, year)
			if err != nil {
				continue
			}
			entries = append(entries, found...)
			time.Sleep(time.Second)
		}
	}

	return entries, nil
}

type nhtsaRecallResponse struct {
	Results []nhtsaRecall `json:"results"`
}

type nhtsaRecall struct {
	NHTSACampaignNumber string `json:"NHTSACampaignNumber"`
	ModelYear           string `json:"ModelYear"`
	Make                string `json:"Make"`
	Model               string `json:"Model"`
	ReportReceivedDate  string `json:"ReportReceivedDate"`
	Summary             string `json:"Summary"`
}

func (s *NHTSASource) fetchRecalls(ctx context.Context, make_ string, year int) ([]graph.ManualEntry, error) {
	u := fmt.Sprintf("https://api.nhtsa.gov/recalls/recallsByVehicle?make=%s&modelYear=%d", make_, year)

	result := fn.Retry(ctx, fn.RetryOpts{
		MaxAttempts: 2,
		InitialWait: time.Second,
		MaxWait:     5 * time.Second,
	}, func(ctx context.Context) fn.Result[[]byte] {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return fn.Err[[]byte](err)
		}
		req.Header.Set("User-Agent", "WessleyBot/1.0")
		resp, err := s.client.Do(req)
		if err != nil {
			return fn.Err[[]byte](err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fn.Errf[[]byte]("status %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		if err != nil {
			return fn.Err[[]byte](err)
		}
		return fn.Ok(body)
	})

	body, err := result.Unwrap()
	if err != nil {
		return nil, err
	}

	var resp nhtsaRecallResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var entries []graph.ManualEntry
	seen := make(map[string]bool)

	for _, recall := range resp.Results {
		if recall.NHTSACampaignNumber == "" {
			continue
		}
		// NHTSA documents can be found via campaign number
		docURL := fmt.Sprintf("https://static.nhtsa.gov/odi/rcl/RCLRPT-%s-0001.PDF", recall.NHTSACampaignNumber)
		if seen[docURL] {
			continue
		}
		seen[docURL] = true

		entries = append(entries, graph.ManualEntry{
			ID:           graph.ManualEntryID(docURL),
			URL:          docURL,
			SourceSite:   "nhtsa.gov",
			Make:         strings.Title(strings.ToLower(recall.Make)),
			Model:        strings.Title(strings.ToLower(recall.Model)),
			Year:         year,
			ManualType:   "service",
			Language:     "en",
			Status:       "discovered",
			DiscoveredAt: time.Now(),
		})
	}

	return entries, nil
}
