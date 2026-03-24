package sink

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// PostgresSink upserts Records into the chunks table with pgvector embeddings.
// The pool is provided externally; lifecycle management (connect/close) is the
// caller's responsibility.
type PostgresSink struct {
	pool *pgxpool.Pool
}

// NewPostgresSink creates a sink backed by an existing pgx pool.
func NewPostgresSink(pool *pgxpool.Pool) *PostgresSink {
	return &PostgresSink{pool: pool}
}

// Name returns the sink name.
func (s *PostgresSink) Name() string { return "PostgresSink" }

// Open is a no-op; the pool is provided at construction time.
func (s *PostgresSink) Open(_ context.Context) error { return nil }

// Write upserts a single Record into the chunks table.
// When Action == ActionDelete, all chunks for the given workspace+file are removed.
func (s *PostgresSink) Write(ctx context.Context, r *core.Record) error {
	if r.Action == core.ActionDelete {
		_, err := s.pool.Exec(ctx,
			`DELETE FROM chunks WHERE workspace_id = $1 AND file_path = $2`,
			r.SourceID, r.Path,
		)
		if err != nil {
			return fmt.Errorf("postgres sink: delete chunks for %s: %w", r.Path, err)
		}
		return nil
	}

	// pgvector requires a typed wrapper; when no embedding is present we store NULL.
	var embedding interface{}
	if len(r.Embedding) > 0 {
		embedding = pgvector.NewVector(r.Embedding)
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO chunks
			(workspace_id, file_path, symbol, symbol_type, language, content, content_hash, embedding, metadata, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		 ON CONFLICT (workspace_id, file_path, symbol)
		 DO UPDATE SET
			symbol_type  = EXCLUDED.symbol_type,
			language     = EXCLUDED.language,
			content      = EXCLUDED.content,
			content_hash = EXCLUDED.content_hash,
			embedding    = EXCLUDED.embedding,
			metadata     = EXCLUDED.metadata,
			updated_at   = NOW()`,
		r.SourceID, r.Path, r.Symbol, r.SymbolType,
		r.Language, r.Content, r.ContentHash, embedding, r.Metadata,
	)
	if err != nil {
		return fmt.Errorf("postgres sink: upsert %s#%s: %w", r.Path, r.Symbol, err)
	}
	return nil
}

// Close is a no-op for PostgresSink; pool lifecycle is managed externally.
func (s *PostgresSink) Close() error { return nil }
