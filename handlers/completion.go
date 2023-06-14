package handlers

import (
	"unicode"
	"unicode/utf8"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
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
	ast.VisitAst(currentAst, visitor)

	table := visitor.Table
	varItems := make(map[string]protocol.CompletionItem)
	aliases := make([]ast.FuncAlias, 0)
	for table != nil {
		for name := range table.Declarations {
			if _, ok := varItems[name]; !ok {
				if fnDecl, ok, isVar := table.LookupDecl(name); ok && isVar {
					varItems[name] = protocol.CompletionItem{
						Kind:  ptr(protocol.CompletionItemKindVariable),
						Label: name,
					}
				} else if table.Enclosing == nil { // functions only in global scope
					aliases = append(aliases, fnDecl.(*ast.FuncDecl).Aliases...)
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
	insertText := ast.TrimStringLit(alias.Original) // remove the ""
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
	table := t.Table
	for table != nil {
		if table == symbols {
			return
		}
		table = table.Enclosing
	}

	t.Table = symbols
	log.Infof("updating scope: %v\n", symbols)
}

func (t *tableVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), t.pos)
}
