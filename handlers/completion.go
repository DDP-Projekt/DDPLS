package handlers

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
	// Add all types
	items := make([]protocol.CompletionItem, 0)
	for _, s := range ddpTypes {
		items = append(items, protocol.CompletionItem{
			Kind:  ptr(protocol.CompletionItemKindClass),
			Label: s,
		})
	}

	// boolean to signify if the next keyword completion should have it's first character Capitalized
	shouldCapitalize := false
	var doc *documents.DocumentState
	// Get the current Document
	if d, ok := documents.Get(params.TextDocument.URI); ok {
		index := params.Position.IndexIn(d.Content) // The index of the cursor
		shouldCapitalize = decideCapitalization(index, d.Content)
		doc = d
	}

	for _, s := range ddpKeywords {
		// Capitalize the first character of the string if it's the start of a sentence
		if shouldCapitalize {
			runes := []rune(s)
			runes[0] = unicode.ToUpper(runes[0])
			s = string(runes)
		}

		items = append(items, protocol.CompletionItem{
			Kind:  ptr(protocol.CompletionItemKindKeyword),
			Label: s,
		})
	}

	currentAst := doc.Module.Ast

	visitor := &tableVisitor{
		Table: currentAst.Symbols,
		pos:   params.Position,
		file:  doc.Module.FileName,
	}

	aliases := make([]ast.FuncAlias, 0)
	for _, stmt := range currentAst.Statements {
		if decl, ok := stmt.(*ast.DeclStmt); ok {
			if funDecl, ok := decl.Decl.(*ast.FuncDecl); ok {
				aliases = append(aliases, funDecl.Aliases...)
			}
		}
	}
	ast.VisitAst(currentAst, visitor)

	ast.VisitAst(currentAst, importVisitor(func(imprt *ast.ImportStmt) {
		contains_name := func(name string) bool {
			for i := range imprt.ImportedSymbols {
				if imprt.ImportedSymbols[i].Literal == name {
					return true
				}
			}
			return false
		}

		is_full_import := len(imprt.ImportedSymbols) == 0

		for name, decl := range imprt.Module.PublicDecls {
			if funDecl, ok := decl.(*ast.FuncDecl); ok && (is_full_import || contains_name(name)) {
				aliases = append(aliases, funDecl.Aliases...)
			}
		}
	}))

	table := visitor.Table
	varItems := make(map[string]protocol.CompletionItem)
	for table != nil {
		for name := range table.Declarations {
			if _, ok := varItems[name]; !ok {
				if _, ok, isVar := table.LookupDecl(name); ok && !isVar {
					continue
				}
				varItems[name] = protocol.CompletionItem{
					Kind:  ptr(protocol.CompletionItemKindVariable),
					Label: name,
				}
			}
		}

		table = table.Enclosing
	}

	for _, v := range varItems {
		items = append(items, v)
	}

	for _, alias := range aliases {
		items = append(items, aliasToCompletionItem(alias))
	}

	return items, nil
}

func decideCapitalization(index int, document string) bool {
	if index-1 == 0 {
		return true
	}

outer:
	// loop backwards and switch on the character at that index
	for i := index - 1; i >= 0; i-- {
		switch r, _ := utf8.DecodeLastRuneInString(document[:i]); r {
		case ' ', '\n', '\r', '\t':
			continue // ignore whitespace
		case ']': // ignore comments
			for bracketCount := 1; i > 0 && bracketCount > 0; i-- {
				if r, _ := utf8.DecodeLastRuneInString(document[:i-1]); r == '[' {
					bracketCount--
					// if comments is at the start of the file
					if i-2 == 0 {
						return true
					}
				} else if r == ']' {
					bracketCount++
				}
			}
		case '.', ':': // start of a new sentence
			return true
		default:
			break outer
		}
	}

	return false
}

func aliasToCompletionItem(alias ast.FuncAlias) protocol.CompletionItem {
	insertText := strings.TrimPrefix(strings.TrimSuffix(alias.Original.Literal, "\""), "\"") // remove the ""
	return protocol.CompletionItem{
		Kind:       ptr(protocol.CompletionItemKindFunction),
		Label:      alias.Func.Name(),
		InsertText: &insertText,
		Detail:     &insertText,
		FilterText: &insertText,
	}
}

func ptr[T any](v T) *T {
	return &v
}

var ddpTypes = []string{
	"Zahl",
	"Kommazahl",
	"Boolean",
	"Text",
	"Buchstabe",
	"Zahlen Liste",
	"Kommazahlen Liste",
	"Boolean Liste",
	"Text Liste",
	"Buchstaben Liste",
}

var ddpKeywords []string

// initialize the ddp-keywords
func init() {
	ddpKeywords = make([]string, 0, len(token.KeywordMap))
	for k := range token.KeywordMap {
		if !helper.Contains(ddpTypes, k) {
			ddpKeywords = append(ddpKeywords, k)
		}
	}
}

type tableVisitor struct {
	Table *ast.SymbolTable
	pos   protocol.Position
	file  string
}

func (*tableVisitor) BaseVisitor() {}

func (t *tableVisitor) UpdateScope(symbols *ast.SymbolTable) {
	t.Table = symbols
}

func (t *tableVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), t.pos)
}

type importVisitor func(imprt *ast.ImportStmt)

func (importVisitor) BaseVisitor() {}

func (f importVisitor) VisitImportStmt(imprt *ast.ImportStmt) {
	f(imprt)
}
