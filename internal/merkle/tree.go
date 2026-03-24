package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// Tree maintains a content-addressable snapshot of file hashes for a workspace.
// It is used by MerkleDedup to skip files that have not changed since the last run.
// The tree is loaded from Postgres at pipeline start and flushed back at the end.
type Tree struct {
	mu      sync.RWMutex
	hashes  map[string]string // file_path → sha256 hex of content
	deleted map[string]bool   // file_path → true if marked for deletion
}

// NewTree returns an empty MerkleTree.
func NewTree() *Tree {
	return &Tree{
		hashes:  make(map[string]string),
		deleted: make(map[string]bool),
	}
}

// HashContent returns the SHA-256 hex digest of the given bytes.
func HashContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// Get returns the stored hash for the given path, and whether it exists.
func (t *Tree) Get(path string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	h, ok := t.hashes[path]
	return h, ok
}

// Set stores or updates the hash for a path.
func (t *Tree) Set(path, hash string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hashes[path] = hash
	delete(t.deleted, path)
}

// MarkDeleted marks a path as deleted in the tree.
func (t *Tree) MarkDeleted(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deleted[path] = true
	delete(t.hashes, path)
}

// IsDeleted returns true if the path has been marked for deletion.
func (t *Tree) IsDeleted(path string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.deleted[path]
}

// Paths returns all tracked file paths (not deleted).
func (t *Tree) Paths() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	paths := make([]string, 0, len(t.hashes))
	for p := range t.hashes {
		paths = append(paths, p)
	}
	return paths
}

// Snapshot returns a copy of the internal hash map for persistence.
func (t *Tree) Snapshot() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	snap := make(map[string]string, len(t.hashes))
	for k, v := range t.hashes {
		snap[k] = v
	}
	return snap
}

// Load replaces the tree contents with the given snapshot (used on startup).
func (t *Tree) Load(hashes map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hashes = make(map[string]string, len(hashes))
	for k, v := range hashes {
		t.hashes[k] = v
	}
	t.deleted = make(map[string]bool)
}
