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
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentCompletion(dm *documents.DocumentManager) protocol.TextDocumentCompletionFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		var docModule *ast.Module
		var latestError *ddperror.Error
		// Get the current Document
		if d, ok := dm.Get(params.TextDocument.URI); ok {
			docModule = d.Module
			for _, err := range d.LatestErrors {
				if helper.IsInRange(err.Range, params.Position) {
					latestError = &err
					break
				}
			}
		}

		// in case of import completion we need nothing else
		importVisitor := &importVisitor{
			pos:               params.Position,
			modPath:           docModule.FileName,
			isSlashCompletion: params.Context.TriggerKind == protocol.CompletionTriggerKindTriggerCharacter && *params.Context.TriggerCharacter == "/",
		}
		ast.VisitModule(docModule, importVisitor)
		if importVisitor.didVisit {
			return importVisitor.items, nil
		}

		visitor := &tableVisitor{
			Table:           docModule.Ast.Symbols,
			tempTable:       docModule.Ast.Symbols,
			pos:             params.Position,
			isDotCompletion: params.Context.TriggerKind == protocol.CompletionTriggerKindTriggerCharacter && *params.Context.TriggerCharacter == ".",
		}
		ast.VisitModule(docModule, visitor)

		items := make([]protocol.CompletionItem, 0, len(ddpTypes)+53)

		// in case of dot completions we don't need anything else
		if visitor.isDotCompletion {
			items = appendDotCompletion(items, visitor.ident, params.Position)
			return items, nil
		}

		items = appendDDPTypes(items)

		table := visitor.Table
		varItems := make(map[string]struct{}, 16)
		wantType := latestError != nil && latestError.Code == ddperror.SYN_EXPECTED_TYPENAME
		for table != nil {
			for name := range table.Declarations {
				decl, _, _ := table.LookupDecl(name)
				if decl.Module() == docModule && decl.GetRange().Start.IsBehind(helper.FromProtocolPosition(params.Position)) {
					continue
				}

				switch decl := decl.(type) {
				case *ast.VarDecl:
					items = appendVarName(items, varItems, decl.Name(), wantType)
				case *ast.FuncDecl:
					for _, a := range decl.Aliases {
						items = appendAlias(items, a, wantType)
					}
				case *ast.StructDecl:
					for _, a := range decl.Aliases {
						items = appendAlias(items, a, wantType)
					}
					items = appendTypeName(items, decl)
				case *ast.TypeAliasDecl:
					items = appendTypeName(items, decl)
				case *ast.TypeDefDecl:
					items = appendTypeName(items, decl)
				}
			}
			table = table.Enclosing
		}

		return items, nil
	})
}

func appendVarName(items []protocol.CompletionItem, varItems map[string]struct{}, name string, wantType bool) []protocol.CompletionItem {
	if _, ok := varItems[name]; !ok && !wantType {
		varItems[name] = struct{}{}
		return append(items, protocol.CompletionItem{
			Kind:  ptr(protocol.CompletionItemKindVariable),
			Label: name,
		})
	}
	return items
}

func appendAlias(items []protocol.CompletionItem, a ast.Alias, wantType bool) []protocol.CompletionItem {
	if wantType {
		return items
	}
	return append(items, aliasToCompletionItem(a)...)
}

func appendTypeName(items []protocol.CompletionItem, decl ast.Declaration) []protocol.CompletionItem {
	return append(items, protocol.CompletionItem{
		Kind:  ptr(protocol.CompletionItemKindClass),
		Label: decl.Name(),
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

func appendDDPTypes(items []protocol.CompletionItem) []protocol.CompletionItem {
	for _, s := range ddpTypes {
		items = append(items, protocol.CompletionItem{
			Kind:  ptr(protocol.CompletionItemKindClass),
			Label: s,
		})
	}
	return items
}

func appendDotCompletion(items []protocol.CompletionItem, ident *ast.Ident, pos protocol.Position) []protocol.CompletionItem {
	if ident == nil || ident.Declaration == nil {
		return items
	}
	if _, isConst := ident.Declaration.(*ast.ConstDecl); isConst {
		return items
	}
	if !ddptypes.IsStruct(ident.Declaration.(*ast.VarDecl).Type) {
		return items
	}

	structType := ident.Declaration.(*ast.VarDecl).Type.(*ddptypes.StructType)
	for _, field := range structType.Fields {
		items = append(items, protocol.CompletionItem{
			Kind:     ptr(protocol.CompletionItemKindField),
			Label:    field.Name,
			SortText: ptr("0"),
			TextEdit: protocol.TextEdit{
				NewText: fmt.Sprintf("%s von %s", field.Name, ident.Declaration.Name()),
				Range: protocol.Range{
					Start: helper.ToProtocolPosition(ident.GetRange().Start),
					End: protocol.Position{
						Line:      pos.Line,
						Character: pos.Character,
					},
				},
			},
			FilterText: ptr(fmt.Sprintf("%s.%s", ident.Declaration.Name(), field.Name)),
		})
	}
	return items
}

type tableVisitor struct {
	Table           *ast.SymbolTable
	tempTable       *ast.SymbolTable
	pos             protocol.Position
	ident           *ast.Ident
	badDecl         *ast.BadDecl
	isDotCompletion bool
}

var (
	_ ast.Visitor            = (*tableVisitor)(nil)
	_ ast.ScopeSetter        = (*tableVisitor)(nil)
	_ ast.ConditionalVisitor = (*tableVisitor)(nil)
	_ ast.IdentVisitor       = (*tableVisitor)(nil)
	_ ast.BadDeclVisitor     = (*tableVisitor)(nil)
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

	pos, end := helper.FromProtocolPosition(t.pos), node.GetRange().End
	if t.isDotCompletion && end.Line == pos.Line && end.Column == pos.Column-1 {
		shouldVisit = true
	}

	return shouldVisit
}

func (t *tableVisitor) VisitIdent(ident *ast.Ident) ast.VisitResult {
	pos, end := helper.FromProtocolPosition(t.pos), ident.GetRange().End
	if t.isDotCompletion && end.Line == pos.Line && end.Column == pos.Column-1 {
		t.ident = ident
	}
	return ast.VisitRecurse
}

func (t *tableVisitor) VisitBadDecl(d *ast.BadDecl) ast.VisitResult {
	t.badDecl = d
	return ast.VisitRecurse
}

type importVisitor struct {
	pos               protocol.Position
	items             []protocol.CompletionItem
	modPath           string
	isSlashCompletion bool
	didVisit          bool
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
		vis.didVisit = true
		// clear the items, because we want no keywords and variables if we
		// are in an import path
		vis.items = make([]protocol.CompletionItem, 0, len(dudenPaths))

		incompletePath := filepath.Dir(ast.TrimStringLit(&imprt.FileName))
		hasDudenPrefix := strings.HasPrefix(incompletePath, "Duden")

		if incompletePath == "." || hasDudenPrefix {
			addDudenPaths(vis.items)
		}

		searchPath := filepath.Join(filepath.Dir(vis.modPath), incompletePath)
		if hasDudenPrefix {
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
				if vis.isSlashCompletion {
					path = incompletePath + "/" + path
				}
				finalPath := strings.TrimPrefix(path, "./")
				finalPath = strings.TrimPrefix(finalPath, ast.TrimStringLit(&imprt.FileName))
				vis.items = append(vis.items, pathToCompletionItem(finalPath))
			}
		}
	}
	return ast.VisitBreak
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

func addDudenPaths(items []protocol.CompletionItem) {
	for _, path := range dudenPaths {
		items = append(items, pathToCompletionItem(path))
	}
}
