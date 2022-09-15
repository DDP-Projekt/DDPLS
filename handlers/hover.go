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

func (h *hoverVisitor) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	if helper.IsInRange(d.InitVal.GetRange(), h.pos) {
		d.InitVal.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if d.Body != nil && helper.IsInRange(d.Body.GetRange(), h.pos) {
		d.Body.Accept(h)
	}
	return h
}

func (h *hoverVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	if decl, ok := h.currentSymbols.LookupVar(e.Literal.Literal); ok {
		val := ""
		if decl.Token().File == h.file {
			val = fmt.Sprintf("[Z %d, S %d]: %s", decl.Name.Line, decl.Name.Column, decl.Type)
		} else {
			datei := h.getHoverFilePath(decl.Name.File)
			val = fmt.Sprintf("[D %s, Z %d, S %d]: %s", datei, decl.Name.Line, decl.Name.Column, decl.Type)
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
	return h
}
func (h *hoverVisitor) VisitIndexing(e *ast.Indexing) ast.Visitor {
	if helper.IsInRange(e.Index.GetRange(), h.pos) {
		return e.Index.Accept(h)
	}
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		return e.Lhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitFloatLit(e *ast.FloatLit) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitCharLit(e *ast.CharLit) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitStringLit(e *ast.StringLit) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitListLit(e *ast.ListLit) ast.Visitor {
	if e.Values != nil {
		for _, expr := range e.Values {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return expr.Accept(h)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if helper.IsInRange(e.Count.GetRange(), h.pos) {
			return e.Count.Accept(h)
		}
		if helper.IsInRange(e.Value.GetRange(), h.pos) {
			return e.Value.Accept(h)
		}
	}
	return h
}
func (h *hoverVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	if helper.IsInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	if helper.IsInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	if helper.IsInRange(e.Mid.GetRange(), h.pos) {
		e.Mid.Accept(h)
	}
	if helper.IsInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	if helper.IsInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	if helper.IsInRange(e.Expr.GetRange(), h.pos) {
		e.Expr.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return expr.Accept(h)
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
			val = fmt.Sprintf("[D %s, Z %d, S %d]\n%s", datei, fun.Token().Line, fun.Token().Column, content[start:end])

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
			val = fmt.Sprintf("[Z %d, S %d]\n%s", fun.Token().Line, fun.Token().Column, h.doc.Content[start:end])

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
	return h
}

func (h *hoverVisitor) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(h)
}
func (h *hoverVisitor) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(h)
}
func (h *hoverVisitor) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	if helper.IsInRange(s.Var.GetRange(), h.pos) {
		return s.Var.Accept(h)
	}
	if helper.IsInRange(s.Rhs.GetRange(), h.pos) {
		return s.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	h.currentSymbols = s.Symbols
	for _, stmt := range s.Statements {
		if helper.IsInRange(stmt.GetRange(), h.pos) {
			return stmt.Accept(h)
		}
	}
	h.currentSymbols = h.currentSymbols.Enclosing
	return h
}
func (h *hoverVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	if helper.IsInRange(s.Condition.GetRange(), h.pos) {
		return s.Condition.Accept(h)
	}
	if helper.IsInRange(s.Then.GetRange(), h.pos) {
		return s.Then.Accept(h)
	}
	if s.Else != nil && helper.IsInRange(s.Else.GetRange(), h.pos) {
		return s.Else.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	if helper.IsInRange(s.Condition.GetRange(), h.pos) {
		return s.Condition.Accept(h)
	}
	if helper.IsInRange(s.Body.GetRange(), h.pos) {
		return s.Body.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), h.pos) {
		return s.Initializer.Accept(h)
	}
	if helper.IsInRange(s.To.GetRange(), h.pos) {
		return s.To.Accept(h)
	}
	if s.StepSize != nil && helper.IsInRange(s.StepSize.GetRange(), h.pos) {
		return s.StepSize.Accept(h)
	}
	if helper.IsInRange(s.Body.GetRange(), h.pos) {
		return s.Body.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), h.pos) {
		return s.Initializer.Accept(h)
	}
	if helper.IsInRange(s.In.GetRange(), h.pos) {
		return s.In.Accept(h)
	}
	if helper.IsInRange(s.Body.GetRange(), h.pos) {
		return s.Body.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(h)
}
func (h *hoverVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value != nil && helper.IsInRange(s.Value.GetRange(), h.pos) {
		return s.Value.Accept(h)
	}
	return h
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
