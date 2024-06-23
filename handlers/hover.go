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

func CreateTextDocumentHover(dm *documents.DocumentManager) protocol.TextDocumentHoverFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
		doc, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("%s not in document map", params.TextDocument.URI)
		}

		hover := &hoverVisitor{
			hover:          nil,
			pos:            params.Position,
			dm:             dm,
			file:           doc.Module.FileName,
			currentSymbols: doc.Module.Ast.Symbols,
			docContent:     doc.Content,
		}

		ast.VisitModule(doc.Module, hover)

		return hover.hover, nil
	})
}

const commentCutset = " \r\n\t[]"

func trimComment(comment *token.Token) string {
	if comment != nil {
		return strings.Trim(comment.Literal, commentCutset)
	}
	return ""
}

type hoverVisitor struct {
	hover          *protocol.Hover
	pos            protocol.Position
	currentSymbols *ast.SymbolTable
	docContent     string
	file           string
	dm             *documents.DocumentManager
}

var (
	_ ast.Visitor            = (*hoverVisitor)(nil)
	_ ast.ConditionalVisitor = (*hoverVisitor)(nil)
	_ ast.ScopeSetter        = (*hoverVisitor)(nil)
	_ ast.ImportStmtVisitor  = (*hoverVisitor)(nil)
)

func (*hoverVisitor) Visitor() {}

func (h *hoverVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), h.pos)
}

func (h *hoverVisitor) SetScope(symbols *ast.SymbolTable) {
	h.currentSymbols = symbols
}

func (h *hoverVisitor) VisitIdent(e *ast.Ident) ast.VisitResult {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {
		header := ""
		if decl.Mod.FileName != h.file {
			header = fmt.Sprintf("%s\n", h.getHoverFilePath(decl.Mod.FileName))
		}
		comment := trimComment(decl.Comment())
		pRange := helper.ToProtocolRange(e.GetRange())
		h.hover = &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf(
					"%s%s\n```ddp\n%s\n```", header, comment, decl.Type,
				),
			},
			Range: &pRange,
		}
	}
	return ast.VisitBreak
}

func (h *hoverVisitor) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return ast.VisitRecurse
			}
		}
	}

	if e.Func == nil {
		return ast.VisitBreak
	}

	// for extern functions we display the whole function,
	// for normal functions only the first line until the colon
	var declRange protocol.Range
	if e.Func.Body != nil {
		declRange = helper.ToProtocolRange(token.NewRange(&e.Func.Tok, &e.Func.Body.Colon))
	} else {
		declRange = helper.ToProtocolRange(e.Func.GetRange())
	}

	moduleContent, is_same_module := h.getDifferentModContent(e.Func.Mod)

	// if the function is in another module, we display the path to that module
	header := ""
	if !is_same_module {
		header = h.getHoverFilePath(e.Func.Mod.FileName)
	}

	start, end := declRange.IndexesIn(moduleContent)
	body := moduleContent[start:end]
	if e.Func.Body != nil {
		body += "\n..."
		endRange := helper.ToProtocolRange(token.Range{
			Start: e.Func.Body.Range.End,
			End:   e.Func.GetRange().End,
		})
		start, end = endRange.IndexesIn(moduleContent)
		body += moduleContent[start:end]
	}

	comment := getCommentDisplayString(e.Func.Comment())

	pRange := helper.ToProtocolRange(e.GetRange())
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("%s\n%s```ddp\n%s\n```", header, comment, body),
		},
		Range: &pRange,
	}

	return ast.VisitBreak
}

func (h *hoverVisitor) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return ast.VisitRecurse
			}
		}
	}

	if e.Struct == nil {
		return ast.VisitBreak
	}

	moduleContent, is_same_module := h.getDifferentModContent(e.Struct.Mod)

	header := ""
	if !is_same_module {
		header = h.getHoverFilePath(e.Struct.Mod.FileName) + "\n"
	}

	declRange := helper.ToProtocolRange(e.Struct.GetRange())
	start, end := declRange.IndexesIn(moduleContent)
	body := moduleContent[start:end]

	comment := getCommentDisplayString(e.Struct.Comment())
	pRange := helper.ToProtocolRange(e.GetRange())
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf(
				"%s%s\n```ddp\n%s\n```", header, comment, body,
			),
		},
		Range: &pRange,
	}
	return ast.VisitBreak
}

// TODO: list all public decls
func (h *hoverVisitor) VisitImportStmt(stmt *ast.ImportStmt) ast.VisitResult {
	if stmt.Module == nil || stmt.Module.Comment == nil {
		return ast.VisitBreak
	}

	comment := trimComment(stmt.Module.Comment)
	pRange := helper.ToProtocolRange(stmt.GetRange())
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindPlainText,
			Value: comment,
		},
		Range: &pRange,
	}
	return ast.VisitBreak
}

func (h *hoverVisitor) getDifferentModContent(mod *ast.Module) (string, bool) {
	is_same_module := mod.FileName == h.file
	// retreive the content of the file in which the function is defined
	moduleContent := h.docContent
	if doc, ok := h.dm.GetFromMod(mod); !is_same_module && ok { // if we already read the file, reuse it
		moduleContent = doc.Content
	} else if !is_same_module { // read the new file
		if content, err := os.ReadFile(mod.FileName); err != nil {
			log.Errorf("Unable to read %s: %s", mod.FileName, err)
		} else {
			moduleContent = string(content)
		}
	}
	return moduleContent, is_same_module
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

// TODO: make comments prettier
func getCommentDisplayString(comment *token.Token) string {
	if comment == nil {
		return ""
	}

	result := trimComment(comment)
	if strings.Contains(result, "\n") {
		return fmt.Sprintf("```ddp\n[\n%s\n]\n```\n", result)
	} else {
		return fmt.Sprintf("```ddp\n[%s]\n```\n", result)
	}
}
