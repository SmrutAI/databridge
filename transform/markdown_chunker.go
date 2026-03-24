package transform

import (
	"context"
	"fmt"
	"strings"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
	"github.com/SmrutAI/ingestion-pipeline/internal/merkle"
)

// MarkdownChunker splits Markdown files into one Record per heading section.
// Non-Markdown records are passed through unchanged.
type MarkdownChunker struct{}

// Name returns the transform name.
func (t *MarkdownChunker) Name() string { return "MarkdownChunker" }

// Apply splits Markdown content at heading boundaries (lines starting with #).
// Each section becomes a separate Record whose Symbol is the heading text.
// If there are no headings the whole document is returned as a single record.
func (t *MarkdownChunker) Apply(_ context.Context, in *core.Record) ([]*core.Record, error) {
	if in.Language != "markdown" {
		return []*core.Record{in}, nil
	}
	sections := splitMarkdown(in.Content)
	if len(sections) == 0 {
		in.Symbol = "_doc"
		in.SymbolType = "section"
		in.ContentHash = merkle.HashContent([]byte(in.Content))
		return []*core.Record{in}, nil
	}
	out := make([]*core.Record, 0, len(sections))
	for i, sec := range sections {
		if strings.TrimSpace(sec.content) == "" {
			continue
		}
		r := &core.Record{
			ID:          fmt.Sprintf("%s#%d", in.ID, i),
			SourceID:    in.SourceID,
			Path:        in.Path,
			Symbol:      sec.heading,
			SymbolType:  "section",
			Language:    "markdown",
			Content:     sec.content,
			ContentHash: merkle.HashContent([]byte(sec.content)),
			Action:      core.ActionUpsert,
			Metadata:    copyMeta(in.Metadata),
		}
		out = append(out, r)
	}
	return out, nil
}

type mdSection struct {
	heading string
	content string
}

// splitMarkdown splits markdown text at heading lines (lines beginning with #).
// Content before the first heading is collected under the synthetic heading "_intro".
func splitMarkdown(text string) []mdSection {
	lines := strings.Split(text, "\n")
	var sections []mdSection
	current := mdSection{heading: "_intro"}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			// Flush the current section before starting a new one.
			if strings.TrimSpace(current.content) != "" {
				sections = append(sections, current)
			}
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			current = mdSection{heading: heading}
		} else {
			current.content += line + "\n"
		}
	}
	if strings.TrimSpace(current.content) != "" {
		sections = append(sections, current)
	}
	return sections
}
