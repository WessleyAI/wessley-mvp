// Package ollama provides an Ollama-backed implementation of the EmbedServiceClient.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"google.golang.org/grpc"
)

// EmbedClient implements mlpb.EmbedServiceClient using Ollama's HTTP API.
type EmbedClient struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewEmbedClient creates an Ollama embedding client.
func NewEmbedClient(baseURL, model string) *EmbedClient {
	return &EmbedClient{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

type ollamaEmbedReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResp struct {
	Embedding []float64 `json:"embedding"`
}

func (c *EmbedClient) embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(ollamaEmbedReq{Model: c.model, Prompt: text})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama embed: status %d", resp.StatusCode)
	}

	var result ollamaEmbedResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama embed decode: %w", err)
	}

	out := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		out[i] = float32(v)
	}
	return out, nil
}

// Embed implements mlpb.EmbedServiceClient.
func (c *EmbedClient) Embed(ctx context.Context, in *mlpb.EmbedRequest, _ ...grpc.CallOption) (*mlpb.EmbedResponse, error) {
	vals, err := c.embed(ctx, in.GetText())
	if err != nil {
		return nil, err
	}
	return &mlpb.EmbedResponse{Values: vals}, nil
}

// EmbedBatch implements mlpb.EmbedServiceClient.
func (c *EmbedClient) EmbedBatch(ctx context.Context, in *mlpb.EmbedBatchRequest, _ ...grpc.CallOption) (*mlpb.EmbedBatchResponse, error) {
	embeddings := make([]*mlpb.EmbedResponse, len(in.GetTexts()))
	for i, text := range in.GetTexts() {
		vals, err := c.embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed batch [%d]: %w", i, err)
		}
		embeddings[i] = &mlpb.EmbedResponse{Values: vals}
	}
	return &mlpb.EmbedBatchResponse{Embeddings: embeddings}, nil
}
