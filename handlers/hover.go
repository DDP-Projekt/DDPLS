package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, ok := documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("%s not in document map", params.TextDocument.URI)
	}

	hover := &hoverVisitor{
		hover:          nil,
		pos:            params.Position,
		currentSymbols: doc.Module.Ast.Symbols,
		doc:            doc,
		file:           doc.Module.FileName,
	}

	ast.VisitAst(doc.Module.Ast, hover)

	return hover.hover, nil
}

const commentCutset = " \r\n\t[]"

func trimComment(comment *token.Token) string {
	result := ""
	if comment != nil {
		result = strings.Trim(comment.Literal, commentCutset) + "\n"
	}
	return result
}

type hoverVisitor struct {
	hover          *protocol.Hover
	pos            protocol.Position
	currentSymbols *ast.SymbolTable
	doc            *documents.DocumentState
	file           string
}

func (*hoverVisitor) BaseVisitor() {}

func (h *hoverVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), h.pos)
}

func (h *hoverVisitor) UpdateScope(symbols *ast.SymbolTable) {
	h.currentSymbols = symbols
}

func (h *hoverVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {
		header := ""
		if decl.Mod.FileName != h.file {
			header = fmt.Sprintf("%s\n", h.getHoverFilePath(decl.Mod.FileName))
		}
		comment := trimComment(decl.Comment)
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
	if fun, ok := e.Func, e.Func != nil; ok {
		var declRange protocol.Range

		if fun.Body != nil {
			declRange = helper.ToProtocolRange(token.NewRange(fun.Tok, fun.Body.Colon))
		} else {
			declRange = helper.ToProtocolRange(fun.GetRange())
		}

		header := ""
		body := ""
		if file := fun.Mod.FileName; file != h.file {
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
		comment := trimComment(fun.Comment)

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
	if strings.HasPrefix(file, ddppath.Duden) {
		datei = "Duden/" + filepath.Base(file)
	}
	return datei
}
