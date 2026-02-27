// Command ingest watches a directory for scraped JSON files and runs them
// through the ingestion pipeline into Qdrant and Neo4j.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/ingest"
	"github.com/WessleyAI/wessley-mvp/engine/scraper"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	"github.com/WessleyAI/wessley-mvp/pkg/fn"
	"github.com/WessleyAI/wessley-mvp/pkg/vehiclenlp"
	"github.com/WessleyAI/wessley-mvp/pkg/metrics"
	"github.com/WessleyAI/wessley-mvp/pkg/ollama"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var met = metrics.New()

// Ingest metrics
var (
	mDocsTotal       = func(source string) *metrics.Counter { return met.Counter(metrics.WithLabels("wessley_ingest_docs_total", "source", source), "Total documents ingested") }
	mErrorsTotal     = func(stage string) *metrics.Counter { return met.Counter(metrics.WithLabels("wessley_ingest_errors_total", "stage", stage), "Total ingestion errors") }
	mDocsSkipped     = met.Counter("wessley_ingest_docs_skipped_total", "Documents skipped by dedup")
	mChunksTotal     = met.Counter("wessley_ingest_chunks_total", "Total chunks created")
	mEmbeddingsTotal = met.Counter("wessley_ingest_embeddings_total", "Total embeddings generated")
	mNeo4jWrites     = met.Counter("wessley_ingest_neo4j_writes_total", "Graph store writes")
	mQdrantWrites    = met.Counter("wessley_ingest_qdrant_writes_total", "Vector store writes")
	mFilesProcessed  = met.Counter("wessley_ingest_files_processed_total", "Files processed")
	mActiveDocs      = met.Gauge("wessley_ingest_active_docs", "Currently processing documents")
	mLastScan        = met.Gauge("wessley_ingest_last_scan_timestamp", "Epoch of last directory scan")
	mPipelineDur     = met.Histogram("wessley_ingest_pipeline_duration_seconds", "Per-doc pipeline time", nil)
	mStageDur        = func(stage string) *metrics.Histogram { return met.Histogram(metrics.WithLabels("wessley_ingest_stage_duration_seconds", "stage", stage), "Per-stage duration", nil) }
	mEmbedDur        = met.Histogram("wessley_ingest_embed_duration_seconds", "Ollama embed call time", nil)

	// Additional metrics
	mBytesProcessed          = met.Counter("wessley_ingest_bytes_processed_total", "Total bytes of source files processed")
	mEmbedBatchSize          = met.Histogram("wessley_ingest_embed_batch_size", "Chunks per embed call", []float64{1, 5, 10, 25, 50, 100, 250, 500})
	mNeo4jDuration           = met.Histogram("wessley_ingest_neo4j_duration_seconds", "Graph write latency", nil)
	mQdrantDuration          = met.Histogram("wessley_ingest_qdrant_duration_seconds", "Vector write latency", nil)
	mDedupHits               = met.Counter("wessley_ingest_dedup_hits_total", "Dedup cache hits")
	mVehicleHierarchyCreated = met.Counter("wessley_ingest_vehicle_hierarchy_created_total", "New vehicle hierarchies created")
	mSystemsDiscovered       = met.Counter("wessley_ingest_systems_discovered_total", "New system nodes created")
	mComponentsExtracted     = met.Counter("wessley_ingest_components_extracted_total", "Components found in content")
	mQueueDepth              = met.Gauge("wessley_ingest_queue_depth", "Files waiting to process")
)

const vectorDims = 768 // nomic-embed-text

func main() {
	var (
		dataDir    = flag.String("dir", "/tmp/wessley-data", "directory to watch for JSON files")
		ollamaURL  = flag.String("ollama", "http://localhost:11434", "Ollama base URL")
		ollamaModel = flag.String("model", "nomic-embed-text", "Ollama embedding model")
		neo4jURL   = flag.String("neo4j", "neo4j://localhost:7687", "Neo4j bolt URL")
		neo4jUser  = flag.String("neo4j-user", "neo4j", "Neo4j username")
		neo4jPass  = flag.String("neo4j-pass", "wessley123", "Neo4j password")
		qdrantAddr = flag.String("qdrant", "localhost:6334", "Qdrant gRPC address")
		collection = flag.String("collection", "wessley", "Qdrant collection name")
		interval   = flag.Duration("interval", 30*time.Second, "scan interval")
		stateFile  = flag.String("state", "/tmp/wessley-data/.ingest-state.json", "processed files state")
	)
	flag.Parse()

	// Start metrics server with runtime collection
	met.CollectRuntime("wessley_ingest", 15*time.Second)
	met.ServeAsync(9091)

	log := slog.Default()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Connect Neo4j
	driver, err := neo4j.NewDriverWithContext(*neo4jURL, neo4j.BasicAuth(*neo4jUser, *neo4jPass, ""))
	if err != nil {
		log.Error("neo4j connect failed", "error", err)
		os.Exit(1)
	}
	defer driver.Close(ctx)
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Error("neo4j verify failed", "error", err)
		os.Exit(1)
	}
	log.Info("connected to Neo4j")

	// Connect Qdrant
	vs, err := semantic.New(*qdrantAddr, *collection)
	if err != nil {
		log.Error("qdrant connect failed", "error", err)
		os.Exit(1)
	}
	defer vs.Close()
	if err := vs.EnsureCollection(ctx, vectorDims); err != nil {
		log.Error("qdrant ensure collection failed", "error", err)
		os.Exit(1)
	}
	log.Info("connected to Qdrant", "collection", *collection, "dims", vectorDims)

	// Ollama embedder
	embedder := ollama.NewEmbedClient(*ollamaURL, *ollamaModel)
	log.Info("using Ollama embeddings", "model", *ollamaModel)

	// Graph store
	gs := graph.New(driver)

	// Dedup map
	var mu sync.Mutex
	seen := make(map[string]bool)

	deps := ingest.Deps{
		Embedder:    embedder,
		VectorStore: vs,
		GraphStore:  gs,
		DeduplicateF: func(_ context.Context, docID string) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			if seen[docID] {
				return true, nil
			}
			seen[docID] = true
			return false, nil
		},
		Logger: log,
	}

	pipeline := ingest.NewPipeline(deps)

	// Load state
	processed := loadState(*stateFile)

	// Ensure data dir
	os.MkdirAll(*dataDir, 0o755)

	log.Info("watching for scraped data", "dir", *dataDir, "interval", *interval)

	scan := func() {
		mLastScan.Set(time.Now().Unix())
		entries, err := os.ReadDir(*dataDir)
		if err != nil {
			mErrorsTotal("scan").Inc()
			log.Error("readdir failed", "error", err)
			return
		}

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name()[0] == '.' {
				continue
			}
			path := filepath.Join(*dataDir, e.Name())
			info, _ := e.Info()
			key := fmt.Sprintf("%s:%d", e.Name(), info.Size())

			if processed[key] {
				continue
			}

			mQueueDepth.Inc()
			log.Info("processing file", "file", e.Name())
			if info != nil {
				mBytesProcessed.Add(info.Size())
			}
			count, errs := processFile(ctx, path, pipeline)
			mQueueDepth.Dec()
			log.Info("file done", "file", e.Name(), "ingested", count, "errors", errs)
			mFilesProcessed.Inc()

			// Only mark as fully processed if no errors (allows retry on next scan)
			if errs == 0 {
				processed[key] = true
				saveState(*stateFile, processed)
			} else {
				log.Warn("file had errors, will retry on next scan", "file", e.Name(), "errors", errs)
			}
		}
	}

	// Initial scan
	scan()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down")
			return
		case <-ticker.C:
			scan()
		}
	}
}

// rawPost is a union of possible scraped formats (Reddit, NHTSA, iFixit, etc.)
type rawPost struct {
	// ScrapedPost fields
	Source      string    `json:"source"`
	SourceID    string    `json:"source_id"`
	Content     string    `json:"content"`
	
	// Reddit raw fields
	ID          string    `json:"id"`
	Subreddit   string    `json:"subreddit"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	SelfText    string    `json:"self_text"`
	URL         string    `json:"url"`
	Permalink   string    `json:"permalink"`
	CreatedUTC  time.Time `json:"created_utc"`
	ScrapedAt   time.Time `json:"scraped_at"`
	
	// NHTSA fields
	ODINumber   string    `json:"odi_number"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Year        int       `json:"year"`
	Summary     string    `json:"summary"`
	Complaint   string    `json:"complaint"`
	DateAdded   string    `json:"date_added"`
	
	// iFixit fields
	GuideID     int       `json:"guide_id"`
	GuideName   string    `json:"guide_name"`
	Category    string    `json:"category"`
	Summary2    string    `json:"summary"` // iFixit uses summary too
}

func (r rawPost) toScrapedPost() scraper.ScrapedPost {
	// Already in ScrapedPost format
	if r.Source != "" && r.SourceID != "" {
		return scraper.ScrapedPost{
			Source:    r.Source,
			SourceID:  r.SourceID,
			Title:     r.Title,
			Content:   r.Content,
			Author:    r.Author,
			URL:       r.URL,
			ScrapedAt: r.ScrapedAt,
		}
	}
	
	// Reddit format
	if r.Subreddit != "" {
		content := r.SelfText
		if content == "" {
			content = r.Title
		}
		return scraper.ScrapedPost{
			Source:      "reddit:" + r.Subreddit,
			SourceID:    r.ID,
			Title:       r.Title,
			Content:     content,
			Author:      r.Author,
			URL:         "https://reddit.com" + r.Permalink,
			PublishedAt: r.CreatedUTC,
			ScrapedAt:   r.ScrapedAt,
		}
	}
	
	// NHTSA format
	if r.ODINumber != "" {
		vehicle := fmt.Sprintf("%d %s %s", r.Year, r.Make, r.Model)
		content := r.Complaint
		if content == "" {
			content = r.Summary
		}
		var vi *scraper.VehicleInfo
		if r.Make != "" && r.Model != "" && r.Year > 0 {
			vi = &scraper.VehicleInfo{
				Make:  r.Make,
				Model: r.Model,
				Year:  r.Year,
			}
		}
		return scraper.ScrapedPost{
			Source:   "nhtsa",
			SourceID: r.ODINumber,
			Title:    vehicle + " - NHTSA Complaint",
			Content:  content,
			URL:      "https://www.nhtsa.gov/",
			Metadata: scraper.Metadata{
				Vehicle:     vehicle,
				VehicleInfo: vi,
				Components:  r.Summary,
			},
		}
	}
	
	// iFixit format
	if r.GuideID != 0 || r.GuideName != "" {
		return scraper.ScrapedPost{
			Source:   "ifixit",
			SourceID: fmt.Sprintf("%d", r.GuideID),
			Title:    r.GuideName,
			Content:  r.Summary2,
		}
	}
	
	return scraper.ScrapedPost{}
}

func processFile(ctx context.Context, path string, pipeline fn.Stage[scraper.ScrapedPost, string]) (int, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 1
	}

	var posts []scraper.ScrapedPost

	// Try decoding as ScrapedPost first (NHTSA, sources scrapers output this format)
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var post scraper.ScrapedPost
		if err := dec.Decode(&post); err != nil {
			break
		}
		// If it decoded as ScrapedPost with content, use it directly
		if post.SourceID != "" && post.Content != "" {
			posts = append(posts, post)
			continue
		}
	}

	// If no posts found, try raw format (Reddit raw output)
	if len(posts) == 0 {
		dec2 := json.NewDecoder(strings.NewReader(string(data)))
		for {
			var raw rawPost
			if err := dec2.Decode(&raw); err != nil {
				break
			}
			post := raw.toScrapedPost()
			if post.SourceID != "" && post.Content != "" {
				posts = append(posts, post)
			}
		}
	}

	// NLP vehicle extraction for posts without structured vehicle info
	for i := range posts {
		if posts[i].Metadata.VehicleInfo == nil {
			text := posts[i].Title + " " + posts[i].Content
			if match := vehiclenlp.ExtractBest(text); match != nil {
				posts[i].Metadata.VehicleInfo = &scraper.VehicleInfo{
					Make:  match.Make,
					Model: match.Model,
					Year:  match.Year,
				}
			}
		}
	}

	count, errs := 0, 0
	log := slog.Default()
	for _, p := range posts {
		if ctx.Err() != nil {
			break
		}
		mActiveDocs.Inc()
		docStart := time.Now()
		result := pipeline(ctx, p)
		mPipelineDur.Since(docStart)
		mActiveDocs.Dec()
		if result.IsErr() {
			_, err := result.Unwrap()
			log.Error("pipeline error", "source_id", p.SourceID, "error", err)
			mErrorsTotal("pipeline").Inc()
			errs++
		} else {
			source := p.Source
			if idx := strings.IndexByte(source, ':'); idx > 0 {
				source = source[:idx]
			}
			mDocsTotal(source).Inc()
			count++
		}
	}
	return count, errs
}

func loadState(path string) map[string]bool {
	m := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	json.Unmarshal(data, &m)
	return m
}

func saveState(path string, m map[string]bool) {
	data, _ := json.Marshal(m)
	os.WriteFile(path, data, 0o644)
}
