package handlers

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentCompletion(dm *documents.DocumentManager) protocol.TextDocumentCompletionFunc {
	return func(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
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
		if d, ok := dm.Get(params.TextDocument.URI); ok {
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

		// must be here at the end because it might clear previous items
		triggerChar := (*string)(nil)
		if params.Context != nil {
			triggerChar = params.Context.TriggerCharacter
		}
		ast.VisitAst(currentAst, &importVisitor{
			pos:         params.Position,
			items:       &items,
			modPath:     doc.Module.FileName,
			triggerChar: triggerChar,
		})

		return items, nil
	}
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

var (
	SupportsSnippets = false // set according to the client capabilities
	aliasRegex       = regexp.MustCompile(`<(\w+)>`)
)

func aliasToCompletionItem(alias ast.FuncAlias) protocol.CompletionItem {
	insertText := ast.TrimStringLit(alias.Original) // remove the ""
	details := insertText
	insertTextMode := protocol.InsertTextFormatPlainText
	if SupportsSnippets {
		insertTextMode = protocol.InsertTextFormatSnippet
		match_count := -1
		insertText = aliasRegex.ReplaceAllStringFunc(insertText, func(b string) string {
			match_count++
			submatches := aliasRegex.FindAllStringSubmatch(insertText, len(alias.Args))
			return fmt.Sprintf("${%d:%s}", match_count+1, submatches[match_count][1])
		})
	}

	documentation := ""
	if alias.Func.Comment != nil {
		documentation = trimComment(alias.Func.Comment)
	}

	return protocol.CompletionItem{
		Kind:             ptr(protocol.CompletionItemKindFunction),
		Documentation:    documentation,
		Label:            alias.Func.Name(),
		InsertText:       &insertText,
		InsertTextFormat: &insertTextMode,
		Detail:           &details,
		FilterText:       &insertText,
	}
}

func ptr[T any](v T) *T {
	return &v
}

var (
	ddpTypes = []string{
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
	ddpKeywords = make([]string, 0, len(token.KeywordMap))
	dudenPaths  = make([]string, 0)
)

func getRelevantEntryName(entry fs.DirEntry) string {
	name := entry.Name()
	if !entry.IsDir() && !strings.HasSuffix(name, ".ddp") {
		return ""
	}
	name = strings.TrimSuffix(name, ".ddp")
	if entry.IsDir() {
		name = name + "/"
	}
	return name
}

// initialize the ddp-keywords
func init() {
	for k := range token.KeywordMap {
		if !helper.Contains(ddpTypes, k) {
			ddpKeywords = append(ddpKeywords, k)
		}
	}
	dudenEntries, err := os.ReadDir(ddppath.Duden)
	if err != nil {
		log.Warningf("Unable to read Duden-Dir: %s", err)
		return
	}
	for _, entry := range dudenEntries {
		if name := getRelevantEntryName(entry); name != "" {
			dudenPaths = append(dudenPaths, "Duden/"+name)
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
}

func (t *tableVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), t.pos)
}

type importVisitor struct {
	pos         protocol.Position
	items       *[]protocol.CompletionItem
	modPath     string
	triggerChar *string
}

func (*importVisitor) BaseVisitor() {}

func (vis *importVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), vis.pos)
}

func (vis *importVisitor) VisitImportStmt(imprt *ast.ImportStmt) {
	if helper.IsInRange(imprt.FileName.Range, protocol.Position(vis.pos)) {
		// clear the items, because we want no keywords and variables if we
		// are in an import path
		*vis.items = make([]protocol.CompletionItem, 0, len(dudenPaths))

		incompletePath := filepath.Dir(ast.TrimStringLit(imprt.FileName))

		if incompletePath == "." {
			addDudenPaths(vis.items)
		}

		searchPath := filepath.Join(filepath.Dir(vis.modPath), incompletePath)
		if vis.triggerChar != nil && *vis.triggerChar == "/" && incompletePath == "Duden" {
			searchPath = ddppath.Duden
		}

		entries, err := os.ReadDir(searchPath)
		if err != nil {
			log.Warningf("unable to read incomplete import Path dir: %s", err)
			return
		}

		modFile := filepath.Base(vis.modPath)
		for _, entry := range entries {
			if !entry.IsDir() && entry.Name() == modFile {
				continue
			}

			if path := getRelevantEntryName(entry); path != "" {
				if vis.triggerChar != nil && *vis.triggerChar != "/" {
					path = incompletePath + "/" + path
				}
				finalPath := strings.TrimPrefix(path, "./")
				*vis.items = append(*vis.items, pathToCompletionItem(finalPath))
			}
		}
	}
	// module could not be parsed yet, return
	if imprt.Module == nil {
		return
	}

	// module could be parsed, complete symbol imports
	for _, ident := range imprt.ImportedSymbols {
		if !helper.IsInRange(ident.Range, vis.pos) {
			continue
		}

		*vis.items = make([]protocol.CompletionItem, 0, len(imprt.Module.PublicDecls))
		for name, decl := range imprt.Module.PublicDecls {
			kind := ptr(protocol.CompletionItemKindFunction)
			if _, ok := decl.(*ast.VarDecl); ok {
				kind = ptr(protocol.CompletionItemKindVariable)
			}
			*vis.items = append(*vis.items, protocol.CompletionItem{
				Kind:  kind,
				Label: name,
			})
		}
		break
	}
}

func pathToCompletionItem(path string) protocol.CompletionItem {
	kind := ptr(protocol.CompletionItemKindFile)
	if strings.HasSuffix(path, "/") {
		kind = ptr(protocol.CompletionItemKindFolder)
	}

	return protocol.CompletionItem{
		Kind:  kind,
		Label: strings.TrimSuffix(path, "/"),
	}
}

func addDudenPaths(items *[]protocol.CompletionItem) {
	for _, path := range dudenPaths {
		*items = append(*items, pathToCompletionItem(path))
	}
}
