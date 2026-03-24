package transform

import (
	"context"
	"fmt"
	"strings"

	"github.com/SmrutAI/databridge/internal/core"
	"github.com/SmrutAI/databridge/internal/merkle"
	"github.com/SmrutAI/databridge/internal/parser"
)

// GoASTParser parses Go source files into one Record per top-level declaration.
// Input records must have Language=="go" and non-empty Content.
// Other language records are passed through unchanged (1:1).
type GoASTParser struct{}

// Name returns the transform name.
func (t *GoASTParser) Name() string { return "GoASTParser" }

// Apply parses the Go file and returns one Record per parsed chunk.
// Non-Go files are returned as-is (single-element slice).
func (t *GoASTParser) Apply(_ context.Context, in *core.Record) ([]*core.Record, error) {
	if in.Language != "go" {
		return []*core.Record{in}, nil
	}
	chunks, err := parser.ParseGo(in.Path, in.Content)
	if err != nil {
		// If parsing fails, return the whole file as a single chunk.
		in.Symbol = "_file"
		in.SymbolType = "file"
		in.ContentHash = merkle.HashContent([]byte(in.Content))
		return []*core.Record{in}, nil
	}
	if len(chunks) == 0 {
		in.Symbol = "_file"
		in.SymbolType = "file"
		in.ContentHash = merkle.HashContent([]byte(in.Content))
		return []*core.Record{in}, nil
	}
	out := make([]*core.Record, 0, len(chunks))
	for i := range chunks {
		chunk := chunks[i]
		if strings.TrimSpace(chunk.Content) == "" {
			continue
		}
		r := &core.Record{
			ID:          fmt.Sprintf("%s#%s", in.ID, chunk.Symbol),
			SourceID:    in.SourceID,
			Path:        in.Path,
			Symbol:      chunk.Symbol,
			SymbolType:  chunk.SymbolType,
			Language:    "go",
			Content:     chunk.Content,
			ContentHash: merkle.HashContent([]byte(chunk.Content)),
			Action:      core.ActionUpsert,
			Metadata:    copyMeta(in.Metadata),
		}
		out = append(out, r)
	}
	return out, nil
}

// copyMeta returns a shallow copy of the metadata map.
// It is shared across all AST parser transforms in this package.
func copyMeta(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
