package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	activeDocument = params.TextDocument.URI
	if err := parse(func(token.Token, string) {}); err != nil {
		log.Errorf("parser error: %s", err)
		return nil, err
	}
	currentAst := currentAst
	doc, _ := getDocument(activeDocument)

	hover := &hoverVisitor{
		hover:          nil,
		pos:            params.Position,
		currentSymbols: currentAst.Symbols,
		doc:            doc,
		file:           currentAst.File,
		varDecls:       make(map[string]*ast.VarDecl),
	}

	for _, stmt := range currentAst.Statements {
		if decl, ok := stmt.(*ast.DeclStmt); ok {
			if varDecl, ok := decl.Decl.(*ast.VarDecl); ok {
				hover.varDecls[varDecl.Name.Literal] = varDecl
			}
		}
		if stmt.Token().File == hover.file && isInRange(stmt.GetRange(), hover.pos) {
			stmt.Accept(hover)
			break
		}
	}

	return hover.hover, nil
}

func isInRange(rang token.Range, pos protocol.Position) bool {
	if pos.Line < uint32(rang.Start.Line-1) || pos.Line > uint32(rang.End.Line-1) {
		return false
	}
	if pos.Line == uint32(rang.Start.Line-1) && pos.Line == uint32(rang.End.Line-1) {
		return pos.Character+1 >= uint32(rang.Start.Column-1) && pos.Character+1 <= uint32(rang.End.Column-1)
	}
	if pos.Line == uint32(rang.Start.Line-1) {
		return pos.Character+1 >= uint32(rang.Start.Column-1)
	}
	if pos.Line == uint32(rang.End.Line-1) {
		return pos.Character+1 <= uint32(rang.End.Column-1)
	}
	return true
}

type hoverVisitor struct {
	hover          *protocol.Hover
	pos            protocol.Position
	currentSymbols *ast.SymbolTable
	doc            *DocumentState
	file           string
	varDecls       map[string]*ast.VarDecl
}

func (h *hoverVisitor) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	if isInRange(d.InitVal.GetRange(), h.pos) {
		d.InitVal.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if d.Body != nil && isInRange(d.Body.GetRange(), h.pos) {
		d.Body.Accept(h)
	}
	return h
}

func (h *hoverVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	if typ, ok := h.currentSymbols.LookupVar(e.Literal.Literal); ok {
		decl, ok := h.varDecls[e.Literal.Literal]
		val := ""
		if ok {
			val = fmt.Sprintf("[Z %d, S %d]: %s", decl.Name.Line, decl.Name.Column, typ.String())
		} else {
			val = typ.String()
		}
		pRange := toProtocolRange(e.GetRange())
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
	if isInRange(e.Index.GetRange(), h.pos) {
		return e.Index.Accept(h)
	}
	if isInRange(e.Lhs.GetRange(), h.pos) {
		return e.Lhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return h
}
func (h *hoverVisitor) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
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
			if isInRange(expr.GetRange(), h.pos) {
				return expr.Accept(h)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if isInRange(e.Count.GetRange(), h.pos) {
			return e.Count.Accept(h)
		}
		if isInRange(e.Value.GetRange(), h.pos) {
			return e.Value.Accept(h)
		}
	}
	return h
}
func (h *hoverVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	if isInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	if isInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	if isInRange(e.Mid.GetRange(), h.pos) {
		e.Mid.Accept(h)
	}
	if isInRange(e.Rhs.GetRange(), h.pos) {
		e.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), h.pos) {
		e.Lhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	if isInRange(e.Expr.GetRange(), h.pos) {
		e.Expr.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if isInRange(expr.GetRange(), h.pos) {
				return expr.Accept(h)
			}
		}
	}
	if fun, ok := h.currentSymbols.LookupFunc(e.Name); ok {
		val := ""
		var declRange protocol.Range
		if fun.Body != nil {
			declRange = toProtocolRange(token.NewRange(fun.Func, fun.Body.Colon))
		} else {
			declRange = toProtocolRange(token.Range{
				Start: token.NewStartPos(fun.Func),
				End: token.Position{
					Line:   fun.ExternFile.Line,
					Column: fun.ExternFile.Column + len(fun.ExternFile.Literal) + len(" definiert"),
				},
			})
		}
		if file := fun.Token().File; file != h.file {
			content, err := os.ReadFile(file)
			if err != nil {
				log.Errorf("Unable to read %s: %s", file, err)
			}
			start, end := declRange.IndexesIn(string(content))

			datei, err := filepath.Rel(h.file, file)
			if err != nil {
				datei = filepath.Base(file)
			} else {
				datei = filepath.ToSlash(strings.TrimPrefix(datei, ".."+string(filepath.Separator)))
			}
			if strings.HasPrefix(file, filepath.Join(scanner.DDPPATH, "Duden")) {
				datei = "Duden/" + filepath.Base(file)
			}

			val = fmt.Sprintf("[D %s, Z %d, S %d]: %s", datei, fun.Token().Line, fun.Token().Column, content[start:end])
		} else {
			start, end := declRange.IndexesIn(h.doc.Content)
			val = fmt.Sprintf("[Z %d, S %d]: %s", fun.Token().Line, fun.Token().Column, h.doc.Content[start:end])
		}

		pRange := toProtocolRange(e.GetRange())
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
	if isInRange(s.Var.GetRange(), h.pos) {
		return s.Var.Accept(h)
	}
	if isInRange(s.Rhs.GetRange(), h.pos) {
		return s.Rhs.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	h.currentSymbols = s.Symbols
	for _, stmt := range s.Statements {
		if decl, ok := stmt.(*ast.DeclStmt); ok {
			if varDecl, ok := decl.Decl.(*ast.VarDecl); ok {
				h.varDecls[varDecl.Name.Literal] = varDecl
			}
		}
		if isInRange(stmt.GetRange(), h.pos) {
			return stmt.Accept(h)
		}
	}
	h.currentSymbols = h.currentSymbols.Enclosing
	return h
}
func (h *hoverVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	if isInRange(s.Condition.GetRange(), h.pos) {
		return s.Condition.Accept(h)
	}
	if isInRange(s.Then.GetRange(), h.pos) {
		return s.Then.Accept(h)
	}
	if s.Else != nil && isInRange(s.Else.GetRange(), h.pos) {
		return s.Else.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	if isInRange(s.Condition.GetRange(), h.pos) {
		return s.Condition.Accept(h)
	}
	if isInRange(s.Body.GetRange(), h.pos) {
		return s.Body.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if isInRange(s.Initializer.GetRange(), h.pos) {
		return s.Initializer.Accept(h)
	}
	if isInRange(s.To.GetRange(), h.pos) {
		return s.To.Accept(h)
	}
	if s.StepSize != nil && isInRange(s.StepSize.GetRange(), h.pos) {
		return s.StepSize.Accept(h)
	}
	if isInRange(s.Body.GetRange(), h.pos) {
		return s.Body.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if isInRange(s.Initializer.GetRange(), h.pos) {
		return s.Initializer.Accept(h)
	}
	if isInRange(s.In.GetRange(), h.pos) {
		return s.In.Accept(h)
	}
	if isInRange(s.Body.GetRange(), h.pos) {
		return s.Body.Accept(h)
	}
	return h
}
func (h *hoverVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(h)
}
func (h *hoverVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value != nil && isInRange(s.Value.GetRange(), h.pos) {
		return s.Value.Accept(h)
	}
	return h
}
