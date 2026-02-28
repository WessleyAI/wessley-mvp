// Package main implements a lightweight RAG chat API for Wessley.
// It embeds questions via Ollama, searches Qdrant, and streams answers.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	"github.com/WessleyAI/wessley-mvp/pkg/ollama"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

const systemPrompt = `You are Wessley, an expert automotive and vehicle knowledge assistant.
Answer the user's question using ONLY the provided context from the knowledge base.
If the context does not contain enough information, say so honestly.
Cite sources when possible. Be concise and helpful.`

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ollamaURL := envOr("OLLAMA_URL", "http://localhost:11434")
	qdrantAddr := envOr("QDRANT_URL", "localhost:6334")
	collection := envOr("QDRANT_COLLECTION", "wessley")
	embedModel := envOr("EMBED_MODEL", "nomic-embed-text")
	chatModel := envOr("CHAT_MODEL", "llama3.1:8b")
	port := envOr("PORT", "8090")

	// Connect Qdrant
	store, err := semantic.New(qdrantAddr, collection)
	if err != nil {
		logger.Error("qdrant connect failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// Ollama embed client
	embedClient := ollama.NewEmbedClient(ollamaURL, embedModel)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, store, embedClient, ollamaURL, chatModel, logger)
	})
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{Addr: ":" + port, Handler: corsMiddleware(mux)}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("chat API starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type chatRequest struct {
	Question string `json:"question"`
}

type sourceDoc struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Source  string  `json:"source"`
	DocID   string  `json:"doc_id"`
	Score   float32 `json:"score"`
}

func handleChat(w http.ResponseWriter, r *http.Request, store *semantic.VectorStore, embedClient *ollama.EmbedClient, ollamaURL, chatModel string, logger *slog.Logger) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Question) == "" {
		http.Error(w, `{"error":"question required"}`, 400)
		return
	}

	ctx := r.Context()

	// 1. Embed the question
	embedResp, err := embedClient.Embed(ctx, &mlpb.EmbedRequest{Text: req.Question})
	if err != nil {
		logger.Error("embed failed", "err", err)
		http.Error(w, `{"error":"embedding failed"}`, 500)
		return
	}

	// 2. Search Qdrant
	results, err := store.Search(ctx, embedResp.GetValues(), 5)
	if err != nil {
		logger.Error("search failed", "err", err)
		http.Error(w, `{"error":"search failed"}`, 500)
		return
	}

	// 3. Build context
	sources := make([]sourceDoc, len(results))
	var contextParts []string
	for i, r := range results {
		sources[i] = sourceDoc{
			ID:      r.ID,
			Content: r.Content,
			Source:  r.Source,
			DocID:   r.DocID,
			Score:   r.Score,
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] (source: %s, score: %.3f)\n%s", i+1, r.Source, r.Score, r.Content))
	}

	contextText := strings.Join(contextParts, "\n\n")
	prompt := fmt.Sprintf("Context from knowledge base:\n%s\n\nUser question: %s", contextText, req.Question)

	// 4. Send sources first, then stream chat
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	// Send sources as first event
	sourcesJSON, _ := json.Marshal(sources)
	fmt.Fprintf(w, "event: sources\ndata: %s\n\n", sourcesJSON)
	flusher.Flush()

	// 5. Stream from Ollama chat
	ollamaReq := map[string]any{
		"model": chatModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"stream": true,
		"options": map[string]any{
			"temperature": 0.3,
		},
	}

	body, _ := json.Marshal(ollamaReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", ollamaURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"request build failed\"}\n\n")
		flusher.Flush()
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"ollama unavailable\"}\n\n")
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Message.Content != "" {
			tokenJSON, _ := json.Marshal(map[string]string{"token": chunk.Message.Content})
			fmt.Fprintf(w, "event: token\ndata: %s\n\n", tokenJSON)
			flusher.Flush()
		}

		if chunk.Done {
			fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			flusher.Flush()
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		logger.Error("stream read error", "err", err)
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}
