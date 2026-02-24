// Package main implements the Wessley API server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	Port        string
	MLWorkerURL string
	Neo4jURL    string
	Neo4jUser   string
	Neo4jPass   string
	QdrantURL   string
	Collection  string
	CORSOrigin  string
}

func loadConfig() Config {
	return Config{
		Port:        envOr("PORT", "8080"),
		MLWorkerURL: envOr("ML_WORKER_URL", "localhost:50051"),
		Neo4jURL:    envOr("NEO4J_URL", "neo4j://localhost:7687"),
		Neo4jUser:   envOr("NEO4J_USER", "neo4j"),
		Neo4jPass:   envOr("NEO4J_PASS", "password"),
		QdrantURL:   envOr("QDRANT_URL", "localhost:6334"),
		Collection:  envOr("QDRANT_COLLECTION", "wessley"),
		CORSOrigin:  envOr("CORS_ORIGIN", "*"),
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
