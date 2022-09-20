package handlers

import (
	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (interface{}, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}

	definition := &definitionVisitor{
		location:       nil,
		currentSymbols: currentAst.Symbols,
		pos:            params.Position,
	}

	for _, stmt := range currentAst.Statements {
		if stmt.Token().File == currentAst.File && helper.IsInRange(stmt.GetRange(), definition.pos) {
			stmt.Accept(definition)
			break
		}
	}

	return definition.location, nil
}

type definitionVisitor struct {
	location       *protocol.Location
	currentSymbols *ast.SymbolTable
	pos            protocol.Position
}

func (def *definitionVisitor) VisitBadDecl(d *ast.BadDecl) {
}
func (def *definitionVisitor) VisitVarDecl(d *ast.VarDecl) {
	if helper.IsInRange(d.InitVal.GetRange(), def.pos) {
		d.InitVal.Accept(def)
	}
}
func (def *definitionVisitor) VisitFuncDecl(d *ast.FuncDecl) {
	if d.Body != nil && helper.IsInRange(d.Body.GetRange(), def.pos) {
		d.Body.Accept(def)
	}
}

func (def *definitionVisitor) VisitBadExpr(e *ast.BadExpr) {
}
func (def *definitionVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := def.currentSymbols.LookupVar(e.Literal.Literal); ok {
		def.location = &protocol.Location{
			URI:   string(uri.FromPath(decl.Token().File)),
			Range: helper.ToProtocolRange(decl.GetRange()),
		}
	}
}
func (def *definitionVisitor) VisitIndexing(e *ast.Indexing) {
	if helper.IsInRange(e.Index.GetRange(), def.pos) {
		e.Index.Accept(def)
		return
	}
	if helper.IsInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
		return
	}
}
func (def *definitionVisitor) VisitIntLit(e *ast.IntLit) {
}
func (def *definitionVisitor) VisitFloatLit(e *ast.FloatLit) {
}
func (def *definitionVisitor) VisitBoolLit(e *ast.BoolLit) {
}
func (def *definitionVisitor) VisitCharLit(e *ast.CharLit) {
}
func (def *definitionVisitor) VisitStringLit(e *ast.StringLit) {
}
func (def *definitionVisitor) VisitListLit(e *ast.ListLit) {
	if e.Values != nil {
		for _, expr := range e.Values {
			if helper.IsInRange(expr.GetRange(), def.pos) {
				expr.Accept(def)
				return
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if helper.IsInRange(e.Count.GetRange(), def.pos) {
			e.Count.Accept(def)
			return
		}
		if helper.IsInRange(e.Value.GetRange(), def.pos) {
			e.Value.Accept(def)
			return
		}
	}
}
func (def *definitionVisitor) VisitUnaryExpr(e *ast.UnaryExpr) {
	if helper.IsInRange(e.Rhs.GetRange(), def.pos) {
		e.Rhs.Accept(def)
	}
}
func (def *definitionVisitor) VisitBinaryExpr(e *ast.BinaryExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
	}
	if helper.IsInRange(e.Rhs.GetRange(), def.pos) {
		e.Rhs.Accept(def)
	}
}
func (def *definitionVisitor) VisitTernaryExpr(e *ast.TernaryExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
	}
	if helper.IsInRange(e.Mid.GetRange(), def.pos) {
		e.Mid.Accept(def)
	}
	if helper.IsInRange(e.Rhs.GetRange(), def.pos) {
		e.Rhs.Accept(def)
	}
}
func (def *definitionVisitor) VisitCastExpr(e *ast.CastExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
	}
}
func (def *definitionVisitor) VisitGrouping(e *ast.Grouping) {
	if helper.IsInRange(e.Expr.GetRange(), def.pos) {
		e.Expr.Accept(def)
	}
}
func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), def.pos) {
				expr.Accept(def)
				return
			}
		}
	}
	if fun, ok := def.currentSymbols.LookupFunc(e.Name); ok {
		def.location = &protocol.Location{
			URI:   string(uri.FromPath(fun.Token().File)),
			Range: helper.ToProtocolRange(fun.GetRange()),
		}
	}
}

func (def *definitionVisitor) VisitBadStmt(s *ast.BadStmt) {
}
func (def *definitionVisitor) VisitDeclStmt(s *ast.DeclStmt) {
	s.Decl.Accept(def)
}
func (def *definitionVisitor) VisitExprStmt(s *ast.ExprStmt) {
	s.Expr.Accept(def)
}
func (def *definitionVisitor) VisitAssignStmt(s *ast.AssignStmt) {
	if helper.IsInRange(s.Var.GetRange(), def.pos) {
		s.Var.Accept(def)
		return
	}
	if helper.IsInRange(s.Rhs.GetRange(), def.pos) {
		s.Rhs.Accept(def)
		return
	}
}
func (def *definitionVisitor) VisitBlockStmt(s *ast.BlockStmt) {
	def.currentSymbols = s.Symbols
	for _, stmt := range s.Statements {
		if helper.IsInRange(stmt.GetRange(), def.pos) {
			stmt.Accept(def)
			return
		}
	}
	def.currentSymbols = def.currentSymbols.Enclosing
}
func (def *definitionVisitor) VisitIfStmt(s *ast.IfStmt) {
	if helper.IsInRange(s.Condition.GetRange(), def.pos) {
		s.Condition.Accept(def)
		return
	}
	if helper.IsInRange(s.Then.GetRange(), def.pos) {
		s.Then.Accept(def)
		return
	}
	if s.Else != nil && helper.IsInRange(s.Else.GetRange(), def.pos) {
		s.Else.Accept(def)
		return
	}
}
func (def *definitionVisitor) VisitWhileStmt(s *ast.WhileStmt) {
	if helper.IsInRange(s.Condition.GetRange(), def.pos) {
		s.Condition.Accept(def)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), def.pos) {
		s.Body.Accept(def)
		return
	}
}
func (def *definitionVisitor) VisitForStmt(s *ast.ForStmt) {
	if helper.IsInRange(s.Initializer.GetRange(), def.pos) {
		s.Initializer.Accept(def)
		return
	}
	if helper.IsInRange(s.To.GetRange(), def.pos) {
		s.To.Accept(def)
		return
	}
	if s.StepSize != nil && helper.IsInRange(s.StepSize.GetRange(), def.pos) {
		s.StepSize.Accept(def)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), def.pos) {
		s.Body.Accept(def)
		return
	}
}
func (def *definitionVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) {
	if helper.IsInRange(s.Initializer.GetRange(), def.pos) {
		s.Initializer.Accept(def)
		return
	}
	if helper.IsInRange(s.In.GetRange(), def.pos) {
		s.In.Accept(def)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), def.pos) {
		s.Body.Accept(def)
		return
	}
}
func (def *definitionVisitor) VisitReturnStmt(s *ast.ReturnStmt) {
	if s.Value != nil && helper.IsInRange(s.Value.GetRange(), def.pos) {
		s.Value.Accept(def)
	}
}
