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
	doc, _ := documents.Get(documents.Active)

	hover := &hoverVisitor{
		hover:          nil,
		pos:            params.Position,
		currentSymbols: currentAst.Symbols,
		doc:            doc,
		file:           currentAst.File,
	}

	for _, stmt := range currentAst.Statements {
		if stmt.Token().File == hover.file && helper.IsInRange(stmt.GetRange(), hover.pos) {
			stmt.Accept(hover)
			break
		}
	}

	return hover.hover, nil
}

type hoverVisitor struct {
	hover          *protocol.Hover
	pos            protocol.Position
	currentSymbols *ast.SymbolTable
	doc            *documents.DocumentState
	file           string
}

func (*hoverVisitor) BaseVisitor() {}

func (h *hoverVisitor) VisitBadDecl(d *ast.BadDecl) {

}
func (h *hoverVisitor) VisitVarDecl(d *ast.VarDecl) {
	if helper.IsInRange(d.InitVal.GetRange(), h.pos) {
		d.InitVal.Accept(h)
	}
}
func (h *hoverVisitor) VisitFuncDecl(d *ast.FuncDecl) {
	if d.Body != nil && helper.IsInRange(d.Body.GetRange(), h.pos) {
		d.Body.Accept(h)
	}
}

func (h *hoverVisitor) VisitBadExpr(e *ast.BadExpr) {

}
func (h *hoverVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := h.currentSymbols.LookupVar(e.Literal.Literal); ok {
		val := ""
		if decl.Token().File == h.file {
			val = fmt.Sprintf("[Z %d, S %d]: %s", decl.Name.Line(), decl.Name.Column(), decl.Type)
		} else {
			datei := h.getHoverFilePath(decl.Name.File)
			val = fmt.Sprintf("[D %s, Z %d, S %d]: %s", datei, decl.Name.Line(), decl.Name.Column(), decl.Type)
		}
		pRange := helper.ToProtocolRange(e.GetRange())
		h.hover = &protocol.Hover{
			Contents: protocol.MarkedStringStruct{
				Language: "ddp",
				Value:    val,
			},
			Range: &pRange,
		}
	}
}
func (h *hoverVisitor) VisitIndexing(e *ast.Indexing) {
	if helper.IsInRange(e.Index.GetRange(), h.pos) {
		e.Index.Accept(h)
		return
	}
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
		return
	}
}
func (h *hoverVisitor) VisitIntLit(e *ast.IntLit) {

}
func (h *hoverVisitor) VisitFloatLit(e *ast.FloatLit) {

}
func (h *hoverVisitor) VisitBoolLit(e *ast.BoolLit) {

}
func (h *hoverVisitor) VisitCharLit(e *ast.CharLit) {

}
func (h *hoverVisitor) VisitStringLit(e *ast.StringLit) {

}
func (h *hoverVisitor) VisitListLit(e *ast.ListLit) {
	if e.Values != nil {
		for _, expr := range e.Values {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				expr.Accept(h)
				return
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if helper.IsInRange(e.Count.GetRange(), h.pos) {
			e.Count.Accept(h)
			return
		}
		if helper.IsInRange(e.Value.GetRange(), h.pos) {
			e.Value.Accept(h)
			return
		}
	}
}
func (h *hoverVisitor) VisitUnaryExpr(e *ast.UnaryExpr) {
	if helper.IsInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
}
func (h *hoverVisitor) VisitBinaryExpr(e *ast.BinaryExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	if helper.IsInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
}
func (h *hoverVisitor) VisitTernaryExpr(e *ast.TernaryExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	if helper.IsInRange(e.Mid.GetRange(), h.pos) {
		e.Mid.Accept(h)
	}
	if helper.IsInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
}
func (h *hoverVisitor) VisitCastExpr(e *ast.CastExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
}
func (h *hoverVisitor) VisitGrouping(e *ast.Grouping) {
	if helper.IsInRange(e.Expr.GetRange(), h.pos) {
		e.Expr.Accept(h)
	}
}
func (h *hoverVisitor) VisitFuncCall(e *ast.FuncCall) {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				expr.Accept(h)
				return
			}
		}
	}
	if fun, ok := h.currentSymbols.LookupFunc(e.Name); ok {
		val := ""
		var declRange protocol.Range

		if fun.Body != nil {
			declRange = helper.ToProtocolRange(token.NewRange(fun.Tok, fun.Body.Colon))
		} else {
			declRange = helper.ToProtocolRange(fun.GetRange())
		}

		if file := fun.Token().File; file != h.file {
			content, err := os.ReadFile(file)
			if err != nil {
				log.Errorf("Unable to read %s: %s", file, err)
			}
			start, end := declRange.IndexesIn(string(content))

			datei := h.getHoverFilePath(file)
			val = fmt.Sprintf("[D %s, Z %d, S %d]\n%s", datei, fun.Token().Line(), fun.Token().Column(), content[start:end])

			if fun.Body != nil {
				val += "\n..."
				endRange := helper.ToProtocolRange(token.Range{
					Start: fun.Body.Range.End,
					End:   fun.GetRange().End,
				})
				start, end = endRange.IndexesIn(string(content))
				val += string(content[start:end])
			}
		} else {
			start, end := declRange.IndexesIn(h.doc.Content)
			val = fmt.Sprintf("[Z %d, S %d]\n%s", fun.Token().Line(), fun.Token().Column(), h.doc.Content[start:end])

			if fun.Body != nil {
				val += "\n..."
				endRange := helper.ToProtocolRange(token.Range{
					Start: fun.Body.Range.End,
					End:   fun.GetRange().End,
				})
				start, end = endRange.IndexesIn(h.doc.Content)
				val += h.doc.Content[start:end]
			}
		}

		pRange := helper.ToProtocolRange(e.GetRange())
		h.hover = &protocol.Hover{
			Contents: protocol.MarkedStringStruct{
				Language: "ddp",
				Value:    val,
			},
			Range: &pRange,
		}
	}
}

func (h *hoverVisitor) VisitBadStmt(s *ast.BadStmt) {

}
func (h *hoverVisitor) VisitDeclStmt(s *ast.DeclStmt) {
	s.Decl.Accept(h)
}
func (h *hoverVisitor) VisitExprStmt(s *ast.ExprStmt) {
	s.Expr.Accept(h)
}
func (h *hoverVisitor) VisitAssignStmt(s *ast.AssignStmt) {
	if helper.IsInRange(s.Var.GetRange(), h.pos) {
		s.Var.Accept(h)
		return
	}
	if helper.IsInRange(s.Rhs.GetRange(), h.pos) {
		s.Rhs.Accept(h)
		return
	}
}
func (h *hoverVisitor) VisitBlockStmt(s *ast.BlockStmt) {
	h.currentSymbols = s.Symbols
	for _, stmt := range s.Statements {
		if helper.IsInRange(stmt.GetRange(), h.pos) {
			stmt.Accept(h)
			return
		}
	}
	h.currentSymbols = h.currentSymbols.Enclosing
}
func (h *hoverVisitor) VisitIfStmt(s *ast.IfStmt) {
	if helper.IsInRange(s.Condition.GetRange(), h.pos) {
		s.Condition.Accept(h)
		return
	}
	if helper.IsInRange(s.Then.GetRange(), h.pos) {
		s.Then.Accept(h)
		return
	}
	if s.Else != nil && helper.IsInRange(s.Else.GetRange(), h.pos) {
		s.Else.Accept(h)
		return
	}
}
func (h *hoverVisitor) VisitWhileStmt(s *ast.WhileStmt) {
	if helper.IsInRange(s.Condition.GetRange(), h.pos) {
		s.Condition.Accept(h)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), h.pos) {
		s.Body.Accept(h)
		return
	}
}
func (h *hoverVisitor) VisitForStmt(s *ast.ForStmt) {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), h.pos) {
		s.Initializer.Accept(h)
		return
	}
	if helper.IsInRange(s.To.GetRange(), h.pos) {
		s.To.Accept(h)
		return
	}
	if s.StepSize != nil && helper.IsInRange(s.StepSize.GetRange(), h.pos) {
		s.StepSize.Accept(h)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), h.pos) {
		s.Body.Accept(h)
		return
	}
}
func (h *hoverVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), h.pos) {
		s.Initializer.Accept(h)
		return
	}
	if helper.IsInRange(s.In.GetRange(), h.pos) {
		s.In.Accept(h)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), h.pos) {
		s.Body.Accept(h)
		return
	}
}
func (h *hoverVisitor) VisitReturnStmt(s *ast.ReturnStmt) {
	if s.Value != nil && helper.IsInRange(s.Value.GetRange(), h.pos) {
		s.Value.Accept(h)
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
