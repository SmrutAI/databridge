package transform

import (
	"context"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
	"github.com/SmrutAI/ingestion-pipeline/internal/merkle"
)

// MerkleDedup skips Records whose content hash matches the stored snapshot,
// avoiding unnecessary re-embedding and re-indexing of unchanged code.
// It updates the tree on every record that passes (new or changed content).
type MerkleDedup struct {
	tree *merkle.Tree
}

// NewMerkleDedup creates a dedup transform backed by the given MerkleTree.
func NewMerkleDedup(tree *merkle.Tree) *MerkleDedup {
	return &MerkleDedup{tree: tree}
}

// Name returns the transform name.
func (t *MerkleDedup) Name() string { return "MerkleDedup" }

// Apply computes the content hash for in.Content and checks it against the tree.
// If the hash is unchanged, it returns a nil slice so the pipeline drops the record.
// If new or changed, it updates the tree and returns the record with ContentHash set.
func (t *MerkleDedup) Apply(_ context.Context, in *core.Record) ([]*core.Record, error) {
	hash := merkle.HashContent([]byte(in.Content))

	if stored, ok := t.tree.Get(in.Path); ok && stored == hash {
		// Content is unchanged — drop the record silently.
		return nil, nil
	}

	// New or changed content — persist the hash and pass the record downstream.
	t.tree.Set(in.Path, hash)
	in.ContentHash = hash
	return []*core.Record{in}, nil
}
