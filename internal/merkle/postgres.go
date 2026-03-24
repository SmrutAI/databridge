package merkle

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadFromPostgres populates the Tree from the merkle_snapshots table.
// workspaceID scopes the query to a single workspace.
func LoadFromPostgres(ctx context.Context, pool *pgxpool.Pool, workspaceID string, tree *Tree) error {
	rows, err := pool.Query(ctx,
		`SELECT file_path, content_hash FROM merkle_snapshots WHERE workspace_id = $1`,
		workspaceID,
	)
	if err != nil {
		return fmt.Errorf("merkle: query snapshots: %w", err)
	}
	defer rows.Close()

	hashes := make(map[string]string)
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return fmt.Errorf("merkle: scan row: %w", err)
		}
		hashes[path] = hash
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("merkle: rows error: %w", err)
	}

	tree.Load(hashes)
	return nil
}

// SaveToPostgres upserts the current tree snapshot into merkle_snapshots.
// Rows with paths no longer in the tree are NOT deleted here — deletion is
// handled per-record by the pipeline when Action == ActionDelete.
func SaveToPostgres(ctx context.Context, pool *pgxpool.Pool, workspaceID string, tree *Tree) error {
	snap := tree.Snapshot()
	if len(snap) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("merkle: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for path, hash := range snap {
		_, err := tx.Exec(ctx,
			`INSERT INTO merkle_snapshots (workspace_id, file_path, content_hash, updated_at)
			 VALUES ($1, $2, $3, NOW())
			 ON CONFLICT (workspace_id, file_path)
			 DO UPDATE SET content_hash = EXCLUDED.content_hash, updated_at = NOW()`,
			workspaceID, path, hash,
		)
		if err != nil {
			return fmt.Errorf("merkle: upsert %s: %w", path, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("merkle: commit: %w", err)
	}
	return nil
}

// DeleteFromPostgres removes a specific file's snapshot from the table.
func DeleteFromPostgres(ctx context.Context, pool *pgxpool.Pool, workspaceID, filePath string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM merkle_snapshots WHERE workspace_id = $1 AND file_path = $2`,
		workspaceID, filePath,
	)
	if err != nil {
		return fmt.Errorf("merkle: delete %s: %w", filePath, err)
	}
	return nil
}
