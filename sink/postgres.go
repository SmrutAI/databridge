package sink

import (
	"context"
	"encoding/json"
	"fmt"

	pgvector "github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/SmrutAI/databridge/internal/core"
)

// PostgresSink upserts Records into the chunks table with pgvector embeddings.
type PostgresSink struct {
	db *gorm.DB
}

// NewPostgresSink creates a sink backed by the given GORM DB.
func NewPostgresSink(db *gorm.DB) *PostgresSink {
	return &PostgresSink{db: db}
}

// Name returns the sink name.
func (s *PostgresSink) Name() string { return "PostgresSink" }

// Open is a no-op; the DB is provided at construction time.
func (s *PostgresSink) Open(_ context.Context) error { return nil }

// Write upserts a single Record into the chunks table.
// When Action == ActionDelete, all chunks for workspace+file are removed.
func (s *PostgresSink) Write(ctx context.Context, r *core.Record) error {
	if r.Action == core.ActionDelete {
		err := s.db.WithContext(ctx).
			Exec(`DELETE FROM chunks WHERE workspace_id = ? AND file_path = ?`,
				r.SourceID, r.Path).Error
		if err != nil {
			return fmt.Errorf("postgres sink: delete chunks for %s: %w", r.Path, err)
		}
		return nil
	}

	metaJSON, err := json.Marshal(r.Metadata)
	if err != nil {
		return fmt.Errorf("postgres sink: marshal metadata for %s#%s: %w", r.Path, r.Symbol, err)
	}

	// pgvector requires a typed wrapper; when no embedding is present we store NULL.
	var embedding any
	if len(r.Embedding) > 0 {
		embedding = pgvector.NewVector(r.Embedding)
	}

	err = s.db.WithContext(ctx).Exec(`
		INSERT INTO chunks
			(workspace_id, file_path, symbol, symbol_type, language, content, content_hash, embedding, metadata, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
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
		r.Language, r.Content, r.ContentHash, embedding, metaJSON,
	).Error
	if err != nil {
		return fmt.Errorf("postgres sink: upsert %s#%s: %w", r.Path, r.Symbol, err)
	}
	return nil
}

// Close is a no-op for PostgresSink; DB lifecycle is managed externally.
func (s *PostgresSink) Close() error { return nil }
