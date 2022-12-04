package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}
	doc, ok := documents.Get(documents.Active)
	if !ok {
		return nil, fmt.Errorf("%s not in document map", documents.Active)
	}

	hover := &hoverVisitor{
		hover:          nil,
		pos:            params.Position,
		currentSymbols: currentAst.Symbols,
		doc:            doc,
		file:           currentAst.File,
	}

	ast.VisitAst(currentAst, hover)

	return hover.hover, nil
}

const commentCutset = " \r\n\t[]"

type hoverVisitor struct {
	hover          *protocol.Hover
	pos            protocol.Position
	currentSymbols *ast.SymbolTable
	doc            *documents.DocumentState
	file           string
}

func (*hoverVisitor) BaseVisitor() {}

func (h *hoverVisitor) UpdateScope(symbols *ast.SymbolTable) {
	h.currentSymbols = symbols
}

func (h *hoverVisitor) ShouldVisit(node ast.Node) bool {
	return node.Token().File == h.file && helper.IsInRange(node.GetRange(), h.pos)
}

func (h *hoverVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := h.currentSymbols.LookupVar(e.Literal.Literal); ok {
		header := ""
		if decl.Token().File != h.file {
			header = fmt.Sprintf("%s\n", h.getHoverFilePath(decl.Name.File))
		}
		comment := ""
		if decl.Comment != nil {
			comment = strings.Trim(decl.Comment.Literal, commentCutset) + "\n"
		}
		pRange := helper.ToProtocolRange(e.GetRange())
		h.hover = &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf(
					"%s%s```ddp\n%s\n```", header, comment, decl.Type,
				),
			},
			Range: &pRange,
		}
	}
}
func (h *hoverVisitor) VisitFuncCall(e *ast.FuncCall) {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return
			}
		}
	}
	if fun, ok := h.currentSymbols.LookupFunc(e.Name); ok {
		var declRange protocol.Range

		if fun.Body != nil {
			declRange = helper.ToProtocolRange(token.NewRange(fun.Tok, fun.Body.Colon))
		} else {
			declRange = helper.ToProtocolRange(fun.GetRange())
		}

		header := ""
		body := ""
		if file := fun.Token().File; file != h.file {
			header = h.getHoverFilePath(file) + "\n"

			content, err := os.ReadFile(file)
			if err != nil {
				log.Errorf("Unable to read %s: %s", file, err)
			}
			start, end := declRange.IndexesIn(string(content))

			body = string(content[start:end])

			if fun.Body != nil {
				body += "\n..."
				endRange := helper.ToProtocolRange(token.Range{
					Start: fun.Body.Range.End,
					End:   fun.GetRange().End,
				})
				start, end = endRange.IndexesIn(string(content))
				body += string(content[start:end])
			}
		} else {
			start, end := declRange.IndexesIn(h.doc.Content)
			body = h.doc.Content[start:end]

			if fun.Body != nil {
				body += "\n..."
				endRange := helper.ToProtocolRange(token.Range{
					Start: fun.Body.Range.End,
					End:   fun.GetRange().End,
				})
				start, end = endRange.IndexesIn(h.doc.Content)
				body += h.doc.Content[start:end]
			}
		}
		comment := ""
		if fun.Comment != nil {
			comment = strings.Trim(fun.Comment.Literal, commentCutset) + "\n"
		}

		pRange := helper.ToProtocolRange(e.GetRange())
		h.hover = &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf("%s\n%s```ddp\n%s\n```", header, comment, body),
			},
			Range: &pRange,
		}
	}
}

// helper to get a nice-looking path to display in a hover
func (h *hoverVisitor) getHoverFilePath(file string) string {
	datei, err := filepath.Rel(h.file, file)
	if err != nil {
		datei = filepath.Base(file)
	} else {
		datei = filepath.ToSlash(strings.TrimPrefix(datei, ".."+string(filepath.Separator)))
	}
	if strings.HasPrefix(file, filepath.Join(scanner.DDPPATH, "Duden")) {
		datei = "Duden/" + filepath.Base(file)
	}
	return datei
}
