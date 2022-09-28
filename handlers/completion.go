package handlers

import (
	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
	"unicode"
	"unicode/utf8"
)

func TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}

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
	// Get the current Document
	if doc, ok := documents.Get(documents.Active); ok {
		index := params.Position.IndexIn(doc.Content) // The index of the cursor
		shouldCapitalize = decideCapitalization(index, doc.Content)
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

	visitor := &tableVisitor{
		Table: currentAst.Symbols,
		pos:   params.Position,
		file:  currentAst.File,
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

	table := visitor.Table
	varItems := make(map[string]protocol.CompletionItem)
	for table != nil {
		for name := range table.Variables {
			if _, ok := varItems[name]; !ok {
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
		Label:      alias.Func,
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
	return node.Token().File == t.file && helper.IsInRange(node.GetRange(), t.pos)
}
