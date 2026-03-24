package transform

import (
	"context"
	"fmt"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// ChunkEmbedder calls an Embedder to populate Record.Embedding for each chunk.
// Records that already carry a non-nil Embedding are passed through unchanged.
type ChunkEmbedder struct {
	embedder core.Embedder
}

// NewChunkEmbedder creates a transform that embeds chunk content using the provided Embedder.
func NewChunkEmbedder(embedder core.Embedder) *ChunkEmbedder {
	return &ChunkEmbedder{embedder: embedder}
}

// Name returns the transform name.
func (t *ChunkEmbedder) Name() string { return "ChunkEmbedder" }

// Apply embeds in.Content using the configured Embedder and stores the result in in.Embedding.
// Records with empty Content or a pre-existing Embedding are returned unchanged.
func (t *ChunkEmbedder) Apply(ctx context.Context, in *core.Record) ([]*core.Record, error) {
	if len(in.Embedding) > 0 {
		return []*core.Record{in}, nil
	}
	if in.Content == "" {
		return []*core.Record{in}, nil
	}
	vec, err := t.embedder.Embed(ctx, in.Content)
	if err != nil {
		return nil, fmt.Errorf("chunk embedder: embed %s#%s: %w", in.Path, in.Symbol, err)
	}
	in.Embedding = vec
	return []*core.Record{in}, nil
}
