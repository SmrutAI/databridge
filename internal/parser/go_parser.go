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
			genChunks := extractGenDeclChunks(d, fset, lines)
			chunks = append(chunks, genChunks...)
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

// extractGenDeclChunks fans out a GenDecl (var/const/type block) into one GoChunk
// per spec, and one per name within each var/const spec. This ensures every symbol
// in a multi-declaration block is indexed independently rather than only the first.
func extractGenDeclChunks(d *ast.GenDecl, fset *token.FileSet, lines []string) []GoChunk {
	var chunks []GoChunk
	for _, spec := range d.Specs {
		specStart := fset.Position(spec.Pos()).Line - 1
		specEnd := fset.Position(spec.End()).Line
		if specEnd > len(lines) {
			specEnd = len(lines)
		}
		content := strings.Join(lines[specStart:specEnd], "\n")
		if strings.TrimSpace(content) == "" {
			continue
		}

		switch d.Tok {
		case token.TYPE:
			if ts, ok := spec.(*ast.TypeSpec); ok {
				chunks = append(chunks, GoChunk{
					Symbol:     ts.Name.Name,
					SymbolType: "type",
					Content:    content,
				})
			}
		case token.VAR, token.CONST:
			if vs, ok := spec.(*ast.ValueSpec); ok {
				symbolType := "var"
				if d.Tok == token.CONST {
					symbolType = "const"
				}
				for _, name := range vs.Names {
					chunks = append(chunks, GoChunk{
						Symbol:     name.Name,
						SymbolType: symbolType,
						Content:    content,
					})
				}
			}
		}
	}
	return chunks
}
