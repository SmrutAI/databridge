package sink

import (
	"context"
	"fmt"
	"os"

	"github.com/SmrutAI/databridge/internal/core"
	smritea "github.com/SmrutAI/smritea-sdk/go"
)

// SmriteaSink writes each Record to smritea as a memory via the smritea Go SDK.
// Config is loaded from environment variables:
//
//	SMRITEA_API_KEY  — required
//	SMRITEA_APP_ID   — required
//	SMRITEA_BASE_URL — optional, default: https://api.smritea.ai
type SmriteaSink struct {
	client *smritea.SmriteaClient
}

// NewSmriteaSink creates a sink using env-configured credentials.
func NewSmriteaSink() (*SmriteaSink, error) {
	apiKey := os.Getenv("SMRITEA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("smritea sink: SMRITEA_API_KEY is required")
	}
	appID := os.Getenv("SMRITEA_APP_ID")
	if appID == "" {
		return nil, fmt.Errorf("smritea sink: SMRITEA_APP_ID is required")
	}
	baseURL := os.Getenv("SMRITEA_BASE_URL")
	client := smritea.NewClient(smritea.ClientConfig{
		APIKey:     apiKey,
		AppID:      appID,
		BaseURL:    baseURL,
		MaxRetries: 2,
	})
	return &SmriteaSink{client: client}, nil
}

// Name returns the sink name.
func (s *SmriteaSink) Name() string { return "SmriteaSink" }

// Open is a no-op for SmriteaSink.
func (s *SmriteaSink) Open(_ context.Context) error { return nil }

// Write sends a single Record to smritea as a memory.
// ActionDelete records are silently skipped (smritea deletion is not addressable by path alone).
func (s *SmriteaSink) Write(ctx context.Context, r *core.Record) error {
	if r.Action == core.ActionDelete {
		return nil
	}
	opts := smritea.NewAddOptions().WithMetadata(map[string]any{
		"workspace_id": r.SourceID,
		"file_path":    r.Path,
		"symbol":       r.Symbol,
		"symbol_type":  r.SymbolType,
		"language":     r.Language,
		"content_hash": r.ContentHash,
		"source":       "codewatch",
	})
	_, err := s.client.Add(ctx, r.Content, opts)
	if err != nil {
		return fmt.Errorf("smritea sink: add memory %s#%s: %w", r.Path, r.Symbol, err)
	}
	return nil
}

// Close is a no-op for SmriteaSink.
func (s *SmriteaSink) Close() error { return nil }
