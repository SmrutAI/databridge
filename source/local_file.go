package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// defaultExtensions is the set of file extensions indexed by default.
var defaultExtensions = map[string]bool{
	".go": true, ".py": true, ".md": true, ".mdx": true,
	".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".rs": true, ".java": true, ".cpp": true, ".c": true, ".h": true,
}

// LocalFileSource walks a local directory and emits one Record per file.
type LocalFileSource struct {
	workspaceID string
	rootDir     string
	extensions  map[string]bool // if nil, uses defaultExtensions
}

// NewLocalFileSource creates a source that walks rootDir for the given workspace.
// If extensions is nil, defaultExtensions are used.
func NewLocalFileSource(workspaceID, rootDir string, extensions map[string]bool) *LocalFileSource {
	if extensions == nil {
		extensions = defaultExtensions
	}
	return &LocalFileSource{
		workspaceID: workspaceID,
		rootDir:     rootDir,
		extensions:  extensions,
	}
}

// Name returns the source name.
func (s *LocalFileSource) Name() string { return "LocalFileSource" }

// Open validates the root directory exists.
func (s *LocalFileSource) Open(_ context.Context) error {
	info, err := os.Stat(s.rootDir)
	if err != nil {
		return fmt.Errorf("local source: stat %s: %w", s.rootDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("local source: %s is not a directory", s.rootDir)
	}
	return nil
}

// Records walks rootDir and sends one Record per matching file.
// The returned channel is closed when the walk completes or ctx is cancelled.
func (s *LocalFileSource) Records(ctx context.Context) (<-chan *core.Record, error) {
	out := make(chan *core.Record, 100)
	go func() {
		defer close(out)
		_ = filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip unreadable entries
			}
			if d.IsDir() {
				// skip hidden directories (e.g. .git, .github)
				if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if !s.extensions[ext] {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil // skip unreadable files
			}
			rel, relErr := filepath.Rel(s.rootDir, path)
			if relErr != nil {
				rel = path
			}
			r := &core.Record{
				ID:       deterministicID(s.workspaceID, rel),
				SourceID: s.workspaceID,
				Path:     rel,
				Language: langFromExt(ext),
				Content:  string(content),
				Action:   core.ActionUpsert,
				Metadata: map[string]any{},
			}
			select {
			case out <- r:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}()
	return out, nil
}

// Close is a no-op for LocalFileSource.
func (s *LocalFileSource) Close() error { return nil }

// langFromExt maps a lowercase file extension to a language identifier.
func langFromExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".md", ".mdx":
		return "markdown"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".h", ".hpp":
		return "c"
	default:
		return "text"
	}
}

// deterministicID returns a stable hex ID derived from workspaceID and path.
// Using SHA-256 ensures the same file in the same workspace always maps to
// the same Qdrant point ID, enabling true upsert semantics across pipeline runs.
func deterministicID(workspaceID, path string) string {
	h := sha256.Sum256([]byte(workspaceID + ":" + path))
	return hex.EncodeToString(h[:])
}
