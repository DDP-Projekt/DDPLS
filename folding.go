package main

import (
	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentFoldingRange(context *glsp.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}

	visitor := &foldingVisitor{
		foldRanges:     nil,
		currentSymbols: currentAst.Symbols,
	}

	for _, stmt := range currentAst.Statements {
		if stmt.Token().File == currentAst.File {
			stmt.Accept(visitor)
		}
	}

	return visitor.foldRanges, nil
}

type foldingVisitor struct {
	foldRanges     []protocol.FoldingRange
	currentSymbols *ast.SymbolTable
}

func (fold *foldingVisitor) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	return d.InitVal.Accept(fold)
}
func (fold *foldingVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if d.Body != nil {
		return d.Body.Accept(fold)
	}
	return fold
}
func (fold *foldingVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitIndexing(e *ast.Indexing) ast.Visitor {
	e.Lhs.Accept(fold)
	return e.Index.Accept(fold)
}
func (fold *foldingVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitFloatLit(e *ast.FloatLit) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitCharLit(e *ast.CharLit) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitStringLit(e *ast.StringLit) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitListLit(e *ast.ListLit) ast.Visitor {
	if e.Values != nil {
		for _, expr := range e.Values {
			expr.Accept(fold)
		}
	} else if e.Count != nil && e.Value != nil {
		e.Count.Accept(fold)
		e.Value.Accept(fold)
	}
	return fold
}
func (fold *foldingVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	return e.Rhs.Accept(fold)
}
func (fold *foldingVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	e.Lhs.Accept(fold)
	return e.Rhs.Accept(fold)
}
func (fold *foldingVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	e.Lhs.Accept(fold)
	e.Mid.Accept(fold)
	return e.Rhs.Accept(fold)
}
func (fold *foldingVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	return e.Lhs.Accept(fold)
}
func (fold *foldingVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(fold)
}
func (fold *foldingVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	for _, arg := range e.Args {
		arg.Accept(fold)
	}
	return fold
}
func (fold *foldingVisitor) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	return fold
}
func (fold *foldingVisitor) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(fold)
}
func (fold *foldingVisitor) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(fold)
}
func (fold *foldingVisitor) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	s.Var.Accept(fold)
	return s.Rhs.Accept(fold)
}
func (fold *foldingVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	fold.currentSymbols = s.Symbols
	for _, stmt := range s.Statements {
		stmt.Accept(fold)
	}
	fold.currentSymbols = fold.currentSymbols.Enclosing

	foldRange := protocol.FoldingRange{
		StartLine: helper.ToProtocolRange(s.GetRange()).Start.Line,
		EndLine:   helper.ToProtocolRange(s.GetRange()).End.Line,
	}

	fold.foldRanges = append(fold.foldRanges, foldRange)

	return fold
}
func (fold *foldingVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	s.Condition.Accept(fold)
	if s.Then != nil {
		s.Then.Accept(fold)
	}
	if s.Else != nil {
		s.Else.Accept(fold)
	}
	return fold
}
func (fold *foldingVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	switch s.While.Type {
	case token.SOLANGE:
		s.Condition.Accept(fold)
		s.Body.Accept(fold)
	case token.MACHE, token.COUNT_MAL:
		s.Body.Accept(fold)
		s.Condition.Accept(fold)
	}
	return fold
}
func (fold *foldingVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	s.Initializer.Accept(fold)
	s.To.Accept(fold)
	if s.StepSize != nil {
		s.StepSize.Accept(fold)
	}
	return s.Body.Accept(fold)
}
func (fold *foldingVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	s.Initializer.Accept(fold)
	s.In.Accept(fold)
	return s.Body.Accept(fold)
}
func (fold *foldingVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(fold)
}
func (fold *foldingVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value == nil {
		return fold
	}
	return s.Value.Accept(fold)
}
