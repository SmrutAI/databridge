package merkle

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/SmrutAI/databridge/internal/store"
)

// LoadFromPostgres populates the Tree from the merkle_snapshots table.
// workspaceID scopes the query to a single workspace.
func LoadFromPostgres(ctx context.Context, db *gorm.DB, workspaceID string, tree *Tree) error {
	var snapshots []store.MerkleSnapshot
	if err := db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Find(&snapshots).Error; err != nil {
		return fmt.Errorf("merkle: query snapshots: %w", err)
	}

	hashes := make(map[string]string, len(snapshots))
	for i := range snapshots {
		hashes[snapshots[i].FilePath] = snapshots[i].ContentHash
	}
	tree.Load(hashes)
	return nil
}

// SaveToPostgres upserts the current tree snapshot into merkle_snapshots.
// Rows with paths no longer in the tree are NOT deleted here — deletion is
// handled per-record by the pipeline when Action == ActionDelete.
func SaveToPostgres(ctx context.Context, db *gorm.DB, workspaceID string, tree *Tree) error {
	snap := tree.Snapshot()
	if len(snap) == 0 {
		return nil
	}

	rows := make([]store.MerkleSnapshot, 0, len(snap))
	for path, hash := range snap {
		rows = append(rows, store.MerkleSnapshot{
			WorkspaceID: workspaceID,
			FilePath:    path,
			ContentHash: hash,
		})
	}

	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "workspace_id"}, {Name: "file_path"}},
			DoUpdates: clause.AssignmentColumns([]string{"content_hash", "updated_at"}),
		}).
		Create(&rows).Error
	if err != nil {
		return fmt.Errorf("merkle: upsert snapshots: %w", err)
	}
	return nil
}

// DeleteFromPostgres removes a specific file's snapshot from the table.
func DeleteFromPostgres(ctx context.Context, db *gorm.DB, workspaceID, filePath string) error {
	err := db.WithContext(ctx).
		Where("workspace_id = ? AND file_path = ?", workspaceID, filePath).
		Delete(&store.MerkleSnapshot{}).Error
	if err != nil {
		return fmt.Errorf("merkle: delete %s: %w", filePath, err)
	}
	return nil
}
