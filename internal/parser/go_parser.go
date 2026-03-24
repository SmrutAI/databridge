package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoChunk represents a single parsed Go symbol (function, method, type, or var/const block).
type GoChunk struct {
	Symbol     string // e.g. "Login", "UserService.Create"
	SymbolType string // "func", "method", "type", "var", "const"
	Content    string // source text of the symbol
}

// ParseGo parses Go source code and returns one GoChunk per top-level declaration.
// src is the full file content as a string.
// filePath is used only for error context.
func ParseGo(filePath, src string) ([]GoChunk, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(src, "\n")
	var chunks []GoChunk

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			chunk := extractFunc(d, fset, lines)
			chunks = append(chunks, chunk)
		case *ast.GenDecl:
			chunk := extractGenDecl(d, fset, lines)
			if chunk.Symbol != "" {
				chunks = append(chunks, chunk)
			}
		}
	}

	return chunks, nil
}

func extractFunc(d *ast.FuncDecl, fset *token.FileSet, lines []string) GoChunk {
	start := fset.Position(d.Pos()).Line - 1
	end := fset.Position(d.End()).Line

	symbolType := "func"
	symbol := d.Name.Name
	if d.Recv != nil && len(d.Recv.List) > 0 {
		symbolType = "method"
		recv := d.Recv.List[0].Type
		var recvName string
		switch r := recv.(type) {
		case *ast.StarExpr:
			if id, ok := r.X.(*ast.Ident); ok {
				recvName = id.Name
			}
		case *ast.Ident:
			recvName = r.Name
		}
		if recvName != "" {
			symbol = recvName + "." + d.Name.Name
		}
	}

	if end > len(lines) {
		end = len(lines)
	}
	content := strings.Join(lines[start:end], "\n")

	return GoChunk{Symbol: symbol, SymbolType: symbolType, Content: content}
}

func extractGenDecl(d *ast.GenDecl, fset *token.FileSet, lines []string) GoChunk {
	start := fset.Position(d.Pos()).Line - 1
	end := fset.Position(d.End()).Line
	if end > len(lines) {
		end = len(lines)
	}
	content := strings.Join(lines[start:end], "\n")

	var symbolType, symbol string
	switch d.Tok {
	case token.TYPE:
		symbolType = "type"
		if len(d.Specs) > 0 {
			if ts, ok := d.Specs[0].(*ast.TypeSpec); ok {
				symbol = ts.Name.Name
			}
		}
	case token.VAR:
		symbolType = "var"
		if len(d.Specs) > 0 {
			if vs, ok := d.Specs[0].(*ast.ValueSpec); ok && len(vs.Names) > 0 {
				symbol = vs.Names[0].Name
			}
		}
	case token.CONST:
		symbolType = "const"
		if len(d.Specs) > 0 {
			if vs, ok := d.Specs[0].(*ast.ValueSpec); ok && len(vs.Names) > 0 {
				symbol = vs.Names[0].Name
			}
		}
	}

	return GoChunk{Symbol: symbol, SymbolType: symbolType, Content: content}
}
