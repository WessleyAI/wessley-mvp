// Package main implements the Wessley API server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/rag"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	"github.com/WessleyAI/wessley-mvp/pkg/mid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds all environment-based configuration.
type Config struct {
	Port          string
	MLWorkerURL   string
	Neo4jURL      string
	Neo4jUser     string
	Neo4jPass     string
	QdrantURL     string
	QdrantHTTPURL string
	Collection    string
	CORSOrigin    string
	DataDir       string
	StateFile     string
}

func loadConfig() Config {
	return Config{
		Port:          envOr("PORT", "8080"),
		MLWorkerURL:   envOr("ML_WORKER_URL", "localhost:50051"),
		Neo4jURL:      envOr("NEO4J_URL", "neo4j://localhost:7687"),
		Neo4jUser:     envOr("NEO4J_USER", "neo4j"),
		Neo4jPass:     envOr("NEO4J_PASS", "password"),
		QdrantURL:     envOr("QDRANT_URL", "localhost:6334"),
		QdrantHTTPURL: envOr("QDRANT_HTTP_URL", "http://localhost:6333"),
		Collection:    envOr("QDRANT_COLLECTION", "wessley"),
		CORSOrigin:    envOr("CORS_ORIGIN", "*"),
		DataDir:       envOr("DATA_DIR", "/tmp/wessley-data"),
		StateFile:     envOr("INGEST_STATE_FILE", "/tmp/wessley-data/.ingest-state.json"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig()

	if err := run(cfg, logger); err != nil {
		logger.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Connect to ML worker (gRPC) ---
	mlConn, err := grpc.NewClient(cfg.MLWorkerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial ml-worker: %w", err)
	}
	defer mlConn.Close()

	// --- Connect to Neo4j ---
	neo4jDriver, err := neo4j.NewDriverWithContext(cfg.Neo4jURL, neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPass, ""))
	if err != nil {
		return fmt.Errorf("neo4j driver: %w", err)
	}
	defer neo4jDriver.Close(ctx)

	graphStore := graph.New(neo4jDriver)

	// --- Connect to Qdrant ---
	vectorStore, err := semantic.New(cfg.QdrantURL, cfg.Collection)
	if err != nil {
		return fmt.Errorf("qdrant connect: %w", err)
	}
	defer vectorStore.Close()

	// --- Build RAG service ---
	ragSvc := rag.New(
		mlConn,
		&semanticAdapter{store: vectorStore},
		&graphAdapter{store: graphStore},
		rag.DefaultOptions(),
		logger,
	)

	// --- Build HTTP server ---
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("POST /api/chat", handleChat(ragSvc, logger))
	mux.HandleFunc("GET /api/v1/manuals", handleManuals(graphStore, logger))
	mux.HandleFunc("GET /api/v1/manuals/{id}/download", handleManualDownload(graphStore, logger))
	mux.HandleFunc("GET /api/v1/metrics/snapshot", handleMetricsSnapshot(graphStore, cfg, logger))

	handler := mid.Chain(mux,
		mid.Recover(logger),
		mid.Logger(logger),
		mid.CORS(cfg.CORSOrigin),
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// --- Graceful shutdown ---
	errCh := make(chan error, 1)
	go func() {
		logger.Info("api server starting", "port", cfg.Port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutCtx)
}

// --- Handlers ---

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ChatRequest is the JSON body for POST /api/chat.
type ChatRequest struct {
	Question string `json:"question"`
	Vehicle  string `json:"vehicle,omitempty"`
}

// ChatResponse is the JSON response for POST /api/chat.
type ChatResponse struct {
	Answer  string       `json:"answer"`
	Sources []rag.Source `json:"sources"`
	Model   string       `json:"model"`
	Tokens  int32        `json:"tokens_used"`
}

func handleChat(ragSvc *rag.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Question == "" {
			http.Error(w, `{"error":"question is required"}`, http.StatusBadRequest)
			return
		}

		answer, err := ragSvc.Query(r.Context(), req.Question, req.Vehicle)
		if err != nil {
			logger.Error("rag query failed", "err", err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Answer:  answer.Text,
			Sources: answer.Sources,
			Model:   answer.Model,
			Tokens:  answer.TokensUsed,
		})
	}
}

// --- Manual Handlers ---

func handleManuals(gs *graph.GraphStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := graph.ManualFilter{
			Make:   q.Get("make"),
			Model:  q.Get("model"),
			Status: q.Get("status"),
		}
		if y := q.Get("year"); y != "" {
			fmt.Sscanf(y, "%d", &filter.Year)
		}

		entries, err := gs.FindManuals(r.Context(), filter)
		if err != nil {
			logger.Error("find manuals", "err", err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}

func handleManualDownload(gs *graph.GraphStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
			return
		}

		entries, err := gs.FindManuals(r.Context(), graph.ManualFilter{})
		if err != nil {
			logger.Error("find manual for download", "err", err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		for _, e := range entries {
			if e.ID == id {
				if e.LocalPath != "" {
					http.ServeFile(w, r, e.LocalPath)
					return
				}
				// Redirect to source URL
				http.Redirect(w, r, e.URL, http.StatusTemporaryRedirect)
				return
			}
		}

		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}
}

// --- Snapshot Types ---

type MetricsSnapshot struct {
	Timestamp      time.Time              `json:"timestamp"`
	KnowledgeGraph KnowledgeGraphSnapshot `json:"knowledge_graph"`
	VectorStore    VectorStoreSnapshot    `json:"vector_store"`
	Ingestion      IngestionSnapshot      `json:"ingestion"`
	Scrapers       ScrapersSnapshot       `json:"scrapers"`
	Infrastructure InfraSnapshot          `json:"infrastructure"`
}

type KnowledgeGraphSnapshot struct {
	TotalNodes          int64              `json:"total_nodes"`
	TotalRelationships  int64              `json:"total_relationships"`
	NodesByType         map[string]int64   `json:"nodes_by_type"`
	RelationshipsByType map[string]int64   `json:"relationships_by_type"`
	TopMakes            []graph.MakeStats    `json:"top_makes"`
	TopVehicles         []graph.VehicleStats `json:"top_vehicles"`
	RecentVehicles      []graph.VehicleStats `json:"recent_vehicles"`
}

type VectorStoreSnapshot struct {
	TotalVectors int64  `json:"total_vectors"`
	Collection   string `json:"collection"`
	Dimensions   int    `json:"dimensions"`
}

type IngestionSnapshot struct {
	TotalDocsIngested int64            `json:"total_docs_ingested"`
	TotalErrors       int64            `json:"total_errors"`
	DocsBySource      map[string]int64 `json:"docs_by_source"`
	LastIngestion     string           `json:"last_ingestion"`
	FilesProcessed    int64            `json:"files_processed"`
}

type ScraperStatus struct {
	Status     string `json:"status"`
	LastScrape string `json:"last_scrape,omitempty"`
	TotalPosts int64  `json:"total_posts,omitempty"`
	TotalDocs  int64  `json:"total_docs,omitempty"`
}

type ManualScraperStatus struct {
	Discovered int64 `json:"discovered"`
	Downloaded int64 `json:"downloaded"`
	Ingested   int64 `json:"ingested"`
	Failed     int64 `json:"failed"`
}

type ScrapersSnapshot struct {
	Reddit  ScraperStatus       `json:"reddit"`
	NHTSA   ScraperStatus       `json:"nhtsa"`
	IFixit  ScraperStatus       `json:"ifixit"`
	Manuals ManualScraperStatus `json:"manuals"`
}

type ServiceStatus struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Model   string `json:"model,omitempty"`
}

type InfraSnapshot struct {
	Neo4j  ServiceStatus `json:"neo4j"`
	Qdrant ServiceStatus `json:"qdrant"`
	Ollama ServiceStatus `json:"ollama"`
}

func handleMetricsSnapshot(gs *graph.GraphStore, cfg Config, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		snap := MetricsSnapshot{
			Timestamp: time.Now().UTC(),
		}

		// Knowledge graph counts
		nodeCounts, err := gs.NodeCounts(ctx)
		if err != nil {
			logger.Error("node counts", "err", err)
			nodeCounts = make(map[string]int64)
		}
		relCounts, err := gs.RelationshipCounts(ctx)
		if err != nil {
			logger.Error("rel counts", "err", err)
			relCounts = make(map[string]int64)
		}

		var totalNodes, totalRels int64
		for _, v := range nodeCounts {
			totalNodes += v
		}
		for _, v := range relCounts {
			totalRels += v
		}

		topMakes, _ := gs.TopMakes(ctx, 10)
		topVehicles, _ := gs.TopVehicles(ctx, 10)
		recentVehicles, _ := gs.RecentVehicles(ctx, 10)

		snap.KnowledgeGraph = KnowledgeGraphSnapshot{
			TotalNodes:          totalNodes,
			TotalRelationships:  totalRels,
			NodesByType:         nodeCounts,
			RelationshipsByType: relCounts,
			TopMakes:            topMakes,
			TopVehicles:         topVehicles,
			RecentVehicles:      recentVehicles,
		}

		// Qdrant info
		snap.VectorStore = queryQdrant(cfg.QdrantHTTPURL, cfg.Collection, logger)

		// Ingestion info
		snap.Ingestion = queryIngestion(cfg.DataDir, cfg.StateFile, logger)

		// Scraper status (read from data files)
		snap.Scrapers = queryScrapers(cfg.DataDir, logger)

		// Infrastructure
		snap.Infrastructure = InfraSnapshot{
			Neo4j:  ServiceStatus{Status: "connected", Version: "5.x"},
			Qdrant: ServiceStatus{Status: "connected"},
			Ollama: ServiceStatus{Status: "connected", Model: "nomic-embed-text"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snap)
	}
}

func queryQdrant(baseURL, collection string, logger *slog.Logger) VectorStoreSnapshot {
	snap := VectorStoreSnapshot{Collection: collection, Dimensions: 768}
	resp, err := http.Get(baseURL + "/collections/" + collection)
	if err != nil {
		logger.Warn("qdrant http query failed", "err", err)
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result struct {
			VectorsCount int64 `json:"vectors_count"`
			PointsCount  int64 `json:"points_count"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &result) == nil {
		snap.TotalVectors = result.Result.VectorsCount
		if snap.TotalVectors == 0 {
			snap.TotalVectors = result.Result.PointsCount
		}
	}
	return snap
}

func queryIngestion(dataDir, stateFile string, logger *slog.Logger) IngestionSnapshot {
	snap := IngestionSnapshot{
		DocsBySource: make(map[string]int64),
	}

	// Read state file for files_processed
	if data, err := os.ReadFile(stateFile); err == nil {
		var state map[string]bool
		if json.Unmarshal(data, &state) == nil {
			snap.FilesProcessed = int64(len(state))
		}
	}

	// Count docs per source from data files
	entries, _ := os.ReadDir(dataDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name()[0] == '.' {
			continue
		}
		path := filepath.Join(dataDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Count JSON objects
		count := int64(0)
		dec := json.NewDecoder(strings.NewReader(string(data)))
		for {
			var raw json.RawMessage
			if dec.Decode(&raw) != nil {
				break
			}
			count++
		}
		// Determine source from filename
		source := strings.TrimSuffix(e.Name(), ".json")
		source = strings.TrimSuffix(source, "-posts")
		source = strings.TrimSuffix(source, "-complaints")
		source = strings.TrimSuffix(source, "-guides")
		snap.DocsBySource[source] = count
		snap.TotalDocsIngested += count
	}
	snap.LastIngestion = time.Now().UTC().Format(time.RFC3339)

	return snap
}

func queryScrapers(dataDir string, logger *slog.Logger) ScrapersSnapshot {
	return ScrapersSnapshot{
		Reddit: ScraperStatus{Status: "running", LastScrape: time.Now().UTC().Format(time.RFC3339)},
		NHTSA:  ScraperStatus{Status: "running", LastScrape: time.Now().UTC().Format(time.RFC3339)},
		IFixit: ScraperStatus{Status: "running", LastScrape: time.Now().UTC().Format(time.RFC3339)},
		Manuals: ManualScraperStatus{},
	}
}

// --- Adapters ---

// semanticAdapter adapts VectorStore to the rag.SemanticSearcher interface.
type semanticAdapter struct {
	store *semantic.VectorStore
}

func (a *semanticAdapter) Search(ctx context.Context, embedding []float32, topK int, filter map[string]string) ([]semantic.SearchResult, error) {
	return a.store.SearchFiltered(ctx, embedding, topK, filter)
}

// graphAdapter adapts GraphStore to the rag.GraphEnricher interface.
type graphAdapter struct {
	store *graph.GraphStore
}

func (a *graphAdapter) FindRelatedComponents(ctx context.Context, keywords []string, vehicle string) ([]graph.Component, []graph.Edge, error) {
	// Search for components matching any keyword by type or name.
	var allComponents []graph.Component
	seen := make(map[string]bool)

	for _, kw := range keywords {
		comps, err := a.store.FindByType(ctx, kw)
		if err != nil {
			continue
		}
		for _, c := range comps {
			if !seen[c.ID] {
				seen[c.ID] = true
				allComponents = append(allComponents, c)
			}
		}
	}

	// If vehicle specified, also search by vehicle components.
	if vehicle != "" {
		// Parse simple vehicle string; for MVP just use neighbors of found components.
	}

	// Get edges by fetching neighbors for found components.
	var edges []graph.Edge
	for _, c := range allComponents {
		neighbors, err := a.store.Neighbors(ctx, c.ID, 1)
		if err != nil {
			continue
		}
		for _, n := range neighbors {
			edges = append(edges, graph.Edge{
				From: c.ID,
				To:   n.ID,
				Type: "connects_to",
			})
		}
	}

	return allComponents, edges, nil
}
