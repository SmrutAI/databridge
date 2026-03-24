package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIEmbedder calls an OpenAI-compatible /v1/embeddings endpoint.
// Use this when you want to delegate embedding to smritea, OpenAI, or any compatible service.
type APIEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	client  *http.Client
}

// NewAPIEmbedder creates an embedder that calls the given base URL.
// dim must match the actual output dimension of the model at the endpoint.
func NewAPIEmbedder(baseURL, apiKey, model string, dim int) *APIEmbedder {
	return &APIEmbedder{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		dim:     dim,
		client:  &http.Client{},
	}
}

type embedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Embed returns the embedding for a single text.
func (e *APIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedBatch sends all texts in a single HTTP request and returns their embeddings.
func (e *APIEmbedder) EmbedBatch(ctx context.Context, texts []string) (_ [][]float32, err error) {
	body, marshalErr := json.Marshal(embedRequest{Input: texts, Model: e.model})
	if marshalErr != nil {
		return nil, fmt.Errorf("api embedder: marshal request: %w", marshalErr)
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if reqErr != nil {
		return nil, fmt.Errorf("api embedder: create request: %w", reqErr)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, doErr := e.client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("api embedder: do request: %w", doErr)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("api embedder: close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		raw, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("api embedder: status %d (body unreadable: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("api embedder: status %d: %s", resp.StatusCode, string(raw))
	}

	var result embedResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("api embedder: decode response: %w", err)
	}

	out := make([][]float32, len(result.Data))
	for i := range result.Data {
		out[result.Data[i].Index] = result.Data[i].Embedding
	}
	return out, nil
}

// Dimension returns the configured embedding dimension.
func (e *APIEmbedder) Dimension() int { return e.dim }

// Close is a no-op for the API embedder.
func (e *APIEmbedder) Close() error { return nil }
