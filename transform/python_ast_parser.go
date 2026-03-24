package transform

import (
	"context"
	"fmt"
	"strings"

	"github.com/SmrutAI/databridge/internal/core"
	"github.com/SmrutAI/databridge/internal/merkle"
	"github.com/SmrutAI/databridge/internal/parser"
)

// PythonASTParser parses Python source files into one Record per top-level function or class.
// Non-Python records are passed through unchanged.
type PythonASTParser struct{}

// Name returns the transform name.
func (t *PythonASTParser) Name() string { return "PythonASTParser" }

// Apply parses the Python file and returns one Record per parsed chunk.
// If parsing fails or produces no chunks, the whole file is returned as a single record.
func (t *PythonASTParser) Apply(ctx context.Context, in *core.Record) ([]*core.Record, error) {
	if in.Language != "python" {
		return []*core.Record{in}, nil
	}
	chunks, err := parser.ParsePython(ctx, in.Content)
	if err != nil || len(chunks) == 0 {
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
			Language:    "python",
			Content:     chunk.Content,
			ContentHash: merkle.HashContent([]byte(chunk.Content)),
			Action:      core.ActionUpsert,
			Metadata:    copyMeta(in.Metadata),
		}
		out = append(out, r)
	}
	return out, nil
}
