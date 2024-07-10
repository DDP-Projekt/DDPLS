package handlers

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentCompletion(dm *documents.DocumentManager) protocol.TextDocumentCompletionFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		// Add all types
		items := make([]protocol.CompletionItem, 0)
		for _, s := range ddpTypes {
			items = append(items, protocol.CompletionItem{
				Kind:  ptr(protocol.CompletionItemKindClass),
				Label: s,
			})
		}

		var docModule *ast.Module
		// Get the current Document
		if d, ok := dm.Get(params.TextDocument.URI); ok {
			docModule = d.Module
		}

		visitor := &tableVisitor{
			Table: docModule.Ast.Symbols,
			tempTable: docModule.Ast.Symbols,
			pos:   params.Position,
		}
		ast.VisitModule(docModule, visitor)

		table := visitor.Table
		varItems := make(map[string]protocol.CompletionItem)
		aliases := make([]ast.Alias, 0)
		for table != nil {
			for name := range table.Declarations {
				if _, ok := varItems[name]; !ok {
					decl, _, _ := table.LookupDecl(name)
					if decl.GetRange().Start.IsBehind(helper.FromProtocolPosition(params.Position)) {
						continue
					}

					switch decl := decl.(type) {
					case *ast.VarDecl:
						varItems[name] = protocol.CompletionItem{
							Kind:  ptr(protocol.CompletionItemKindVariable),
							Label: name,
						}
					case *ast.FuncDecl:
						for _, a := range decl.Aliases {
							aliases = append(aliases, a)
						}
					case *ast.StructDecl:
						for _, a := range decl.Aliases {
							aliases = append(aliases, a)
						}
					}
				}
			}

			table = table.Enclosing
		}

		for _, v := range varItems {
			items = append(items, v)
		}

		for _, alias := range aliases {
			items = append(items, aliasToCompletionItem(alias)...)
		}

		// must be here at the end because it might clear previous items
		triggerChar := (*string)(nil)
		if params.Context != nil {
			triggerChar = params.Context.TriggerCharacter
		}
		ast.VisitModule(docModule, &importVisitor{
			pos:         params.Position,
			items:       &items,
			modPath:     docModule.FileName,
			triggerChar: triggerChar,
		})

		return items, nil
	})
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

func aliasToCompletionItem(alias ast.Alias) []protocol.CompletionItem {
	orig := alias.GetOriginal()
	insertText := ast.TrimStringLit(&orig) // remove the ""
	details := insertText
	insertTextMode := protocol.InsertTextFormatPlainText
	if SupportsSnippets {
		insertTextMode = protocol.InsertTextFormatSnippet
		match_count := -1
		insertText = aliasRegex.ReplaceAllStringFunc(insertText, func(b string) string {
			match_count++
			submatches := aliasRegex.FindAllStringSubmatch(insertText, len(alias.GetArgs()))
			return fmt.Sprintf("${%d:%s}", match_count+1, submatches[match_count][1])
		})
	}

	documentation := ""
	if comment := alias.Decl().Comment(); comment != nil {
		documentation = trimComment(comment)
	}

	name := alias.Decl().Name()
	return []protocol.CompletionItem{
		{
			Kind:             ptr(protocol.CompletionItemKindFunction),
			Documentation:    documentation,
			Label:            name,
			InsertText:       &insertText,
			InsertTextFormat: &insertTextMode,
			Detail:           &details,
			FilterText:       &insertText,
		},
		{
			Kind:             ptr(protocol.CompletionItemKindFunction),
			Documentation:    documentation,
			Label:            name,
			InsertText:       &insertText,
			InsertTextFormat: &insertTextMode,
			Detail:           &details,
			FilterText:       &name,
		},
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
		"nichts",
	}
	dudenPaths = make([]string, 0)
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
	Table     *ast.SymbolTable
	tempTable *ast.SymbolTable
	pos       protocol.Position
}

var (
	_ ast.Visitor            = (*tableVisitor)(nil)
	_ ast.ScopeSetter        = (*tableVisitor)(nil)
	_ ast.ConditionalVisitor = (*tableVisitor)(nil)
)

func (*tableVisitor) Visitor() {}

func (t *tableVisitor) SetScope(symbols *ast.SymbolTable) {
	t.tempTable = symbols
}

func (t *tableVisitor) ShouldVisit(node ast.Node) bool {
	shouldVisit := helper.IsInRange(node.GetRange(), t.pos)
	if shouldVisit {
		t.Table = t.tempTable
	}
	return shouldVisit
}

type importVisitor struct {
	pos         protocol.Position
	items       *[]protocol.CompletionItem
	modPath     string
	triggerChar *string
}

var (
	_ ast.Visitor            = (*importVisitor)(nil)
	_ ast.ConditionalVisitor = (*importVisitor)(nil)
	_ ast.ImportStmtVisitor  = (*importVisitor)(nil)
)

func (*importVisitor) Visitor() {}

func (vis *importVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), vis.pos)
}

func (vis *importVisitor) VisitImportStmt(imprt *ast.ImportStmt) ast.VisitResult {
	if helper.IsInRange(imprt.FileName.Range, protocol.Position(vis.pos)) {
		// clear the items, because we want no keywords and variables if we
		// are in an import path
		*vis.items = make([]protocol.CompletionItem, 0, len(dudenPaths))

		incompletePath := filepath.Dir(ast.TrimStringLit(&imprt.FileName))

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
			return ast.VisitRecurse
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
		return ast.VisitRecurse
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
	return ast.VisitRecurse
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
