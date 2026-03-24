package parser

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// PythonChunk represents a single parsed Python symbol (function or class).
type PythonChunk struct {
	Symbol     string // function or class name
	SymbolType string // "func" or "class"
	Content    string // source text of the symbol
}

// ParsePython parses Python source code using tree-sitter and returns one
// PythonChunk per top-level function or class definition.
// src is the full file content as a string.
func ParsePython(ctx context.Context, src string) ([]PythonChunk, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	srcBytes := []byte(src)
	tree, err := parser.ParseCtx(ctx, nil, srcBytes)
	if err != nil {
		return nil, fmt.Errorf("python parser: parse: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	var chunks []PythonChunk

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "function_definition":
			chunk := extractPythonFunc(child, srcBytes)
			chunks = append(chunks, chunk)
		case "class_definition":
			chunk := extractPythonClass(child, srcBytes)
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}

func extractPythonFunc(node *sitter.Node, src []byte) PythonChunk {
	nameNode := node.ChildByFieldName("name")
	symbol := ""
	if nameNode != nil {
		symbol = nameNode.Content(src)
	}
	content := node.Content(src)
	return PythonChunk{Symbol: symbol, SymbolType: "func", Content: content}
}

func extractPythonClass(node *sitter.Node, src []byte) PythonChunk {
	nameNode := node.ChildByFieldName("name")
	symbol := ""
	if nameNode != nil {
		symbol = nameNode.Content(src)
	}

	// For classes, include the full class body (name + all methods + body).
	var sb strings.Builder
	sb.WriteString(node.Content(src))
	content := sb.String()

	return PythonChunk{Symbol: symbol, SymbolType: "class", Content: content}
}
