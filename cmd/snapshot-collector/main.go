// Command snapshot-collector fetches a metrics snapshot from the API,
// computes deltas, and writes JSON files for the GitHub Pages dashboard.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Delta represents changes between two consecutive snapshots.
type Delta struct {
	Timestamp    time.Time        `json:"timestamp"`
	Period       string           `json:"period"`
	NewDocs      int64            `json:"new_docs"`
	NewNodes     int64            `json:"new_nodes"`
	NewRelations int64            `json:"new_relations"`
	NewVectors   int64            `json:"new_vectors"`
	ErrorsDelta  int64            `json:"errors_delta"`
	DocsBySource map[string]int64 `json:"docs_by_source"`
	NewVehicles  []string         `json:"new_vehicles"`
}

// Snapshot mirrors the API response (partial, for delta computation).
type Snapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	KnowledgeGraph struct {
		TotalNodes         int64 `json:"total_nodes"`
		TotalRelationships int64 `json:"total_relationships"`
		RecentVehicles     []struct {
			Vehicle string `json:"vehicle"`
		} `json:"recent_vehicles"`
	} `json:"knowledge_graph"`
	VectorStore struct {
		TotalVectors int64 `json:"total_vectors"`
	} `json:"vector_store"`
	Ingestion struct {
		TotalDocsIngested int64            `json:"total_docs_ingested"`
		TotalErrors       int64            `json:"total_errors"`
		DocsBySource      map[string]int64 `json:"docs_by_source"`
	} `json:"ingestion"`
}

const maxHistory = 288

func main() {
	apiURL := flag.String("api", "http://localhost:8080", "API base URL")
	docsDir := flag.String("docs-dir", "docs", "docs directory for output")
	push := flag.Bool("push", false, "git commit and push after update")
	flag.Parse()

	dataDir := filepath.Join(*docsDir, "data")
	os.MkdirAll(dataDir, 0o755)

	latestPath := filepath.Join(dataDir, "metrics-latest.json")
	historyPath := filepath.Join(dataDir, "metrics-history.json")
	prevPath := filepath.Join(dataDir, ".metrics-prev.json")

	// Fetch snapshot from API
	resp, err := http.Get(*apiURL + "/api/v1/metrics/snapshot")
	if err != nil {
		log.Fatalf("fetch snapshot: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("API returned %d: %s", resp.StatusCode, body)
	}

	// Parse current snapshot
	var current Snapshot
	if err := json.Unmarshal(body, &current); err != nil {
		log.Fatalf("parse snapshot: %v", err)
	}

	// Load previous snapshot for delta
	var prev Snapshot
	if data, err := os.ReadFile(prevPath); err == nil {
		json.Unmarshal(data, &prev)
	}

	// Compute delta
	delta := Delta{
		Timestamp:    current.Timestamp,
		Period:       "5m",
		NewDocs:      current.Ingestion.TotalDocsIngested - prev.Ingestion.TotalDocsIngested,
		NewNodes:     current.KnowledgeGraph.TotalNodes - prev.KnowledgeGraph.TotalNodes,
		NewRelations: current.KnowledgeGraph.TotalRelationships - prev.KnowledgeGraph.TotalRelationships,
		NewVectors:   current.VectorStore.TotalVectors - prev.VectorStore.TotalVectors,
		ErrorsDelta:  current.Ingestion.TotalErrors - prev.Ingestion.TotalErrors,
		DocsBySource: make(map[string]int64),
	}

	for k, v := range current.Ingestion.DocsBySource {
		delta.DocsBySource[k] = v - prev.Ingestion.DocsBySource[k]
	}

	// Find new vehicles
	prevVehicles := make(map[string]bool)
	for _, v := range prev.KnowledgeGraph.RecentVehicles {
		prevVehicles[v.Vehicle] = true
	}
	for _, v := range current.KnowledgeGraph.RecentVehicles {
		if !prevVehicles[v.Vehicle] {
			delta.NewVehicles = append(delta.NewVehicles, v.Vehicle)
		}
	}

	// Write latest
	if err := os.WriteFile(latestPath, body, 0o644); err != nil {
		log.Fatalf("write latest: %v", err)
	}

	// Update history
	var history []Delta
	if data, err := os.ReadFile(historyPath); err == nil {
		json.Unmarshal(data, &history)
	}
	history = append(history, delta)
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	histData, _ := json.MarshalIndent(history, "", "  ")
	os.WriteFile(historyPath, histData, 0o644)

	// Save current as prev
	os.WriteFile(prevPath, body, 0o644)

	fmt.Printf("Snapshot collected at %s (docs: %d, nodes: %d, vectors: %d)\n",
		current.Timestamp.Format(time.RFC3339),
		current.Ingestion.TotalDocsIngested,
		current.KnowledgeGraph.TotalNodes,
		current.VectorStore.TotalVectors)
	fmt.Printf("Delta: +%d docs, +%d nodes, +%d rels, +%d vectors\n",
		delta.NewDocs, delta.NewNodes, delta.NewRelations, delta.NewVectors)

	// Git commit + push
	if *push {
		gitCommitPush(*docsDir)
	}
}

func gitCommitPush(docsDir string) {
	cmds := [][]string{
		{"git", "add", filepath.Join(docsDir, "data/")},
		{"git", "commit", "-m", fmt.Sprintf("metrics: snapshot %s", time.Now().UTC().Format("2006-01-02T15:04"))},
		{"git", "push"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("git %v: %v", args, err)
		}
	}
}
