package main

import (
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (interface{}, error) {
	activeDocument = params.TextDocument.URI
	if err := parse(func(token.Token, string) {}); err != nil {
		log.Errorf("parser error: %s", err)
		return nil, err
	}
	currentAst := currentAst

	definition := &definitionVisitor{
		location:       nil,
		currentSymbols: currentAst.Symbols,
		pos:            params.Position,
	}

	for _, stmt := range currentAst.Statements {
		if stmt.Token().File == currentAst.File && isInRange(stmt.GetRange(), definition.pos) {
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

func (def *definitionVisitor) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	if isInRange(d.InitVal.GetRange(), def.pos) {
		d.InitVal.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if d.Body != nil && isInRange(d.Body.GetRange(), def.pos) {
		d.Body.Accept(def)
	}
	return def
}

func (def *definitionVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	if decl, ok := def.currentSymbols.LookupVar(e.Literal.Literal); ok {
		def.location = &protocol.Location{
			URI:   string(URIFromPath(decl.Token().File)),
			Range: toProtocolRange(decl.GetRange()),
		}
	}
	return def
}
func (def *definitionVisitor) VisitIndexing(e *ast.Indexing) ast.Visitor {
	if isInRange(e.Index.GetRange(), def.pos) {
		return e.Index.Accept(def)
	}
	if isInRange(e.Lhs.GetRange(), def.pos) {
		return e.Lhs.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitCharLit(e *ast.CharLit) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitStringLit(e *ast.StringLit) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitListLit(e *ast.ListLit) ast.Visitor {
	if e.Values != nil {
		for _, expr := range e.Values {
			if isInRange(expr.GetRange(), def.pos) {
				return expr.Accept(def)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if isInRange(e.Count.GetRange(), def.pos) {
			return e.Count.Accept(def)
		}
		if isInRange(e.Value.GetRange(), def.pos) {
			return e.Value.Accept(def)
		}
	}
	return def
}
func (def *definitionVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	if isInRange(e.Rhs.GetRange(), def.pos) {
		e.Rhs.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
	}
	if isInRange(e.Rhs.GetRange(), def.pos) {
		e.Rhs.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
	}
	if isInRange(e.Mid.GetRange(), def.pos) {
		e.Mid.Accept(def)
	}
	if isInRange(e.Rhs.GetRange(), def.pos) {
		e.Rhs.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), def.pos) {
		e.Lhs.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	if isInRange(e.Expr.GetRange(), def.pos) {
		e.Expr.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if isInRange(expr.GetRange(), def.pos) {
				return expr.Accept(def)
			}
		}
	}
	if fun, ok := def.currentSymbols.LookupFunc(e.Name); ok {
		def.location = &protocol.Location{
			URI:   string(URIFromPath(fun.Token().File)),
			Range: toProtocolRange(fun.GetRange()),
		}
	}
	return def
}

func (def *definitionVisitor) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	return def
}
func (def *definitionVisitor) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(def)
}
func (def *definitionVisitor) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(def)
}
func (def *definitionVisitor) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	if isInRange(s.Var.GetRange(), def.pos) {
		return s.Var.Accept(def)
	}
	if isInRange(s.Rhs.GetRange(), def.pos) {
		return s.Rhs.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	def.currentSymbols = s.Symbols
	for _, stmt := range s.Statements {
		if isInRange(stmt.GetRange(), def.pos) {
			return stmt.Accept(def)
		}
	}
	def.currentSymbols = def.currentSymbols.Enclosing
	return def
}
func (def *definitionVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	if isInRange(s.Condition.GetRange(), def.pos) {
		return s.Condition.Accept(def)
	}
	if isInRange(s.Then.GetRange(), def.pos) {
		return s.Then.Accept(def)
	}
	if s.Else != nil && isInRange(s.Else.GetRange(), def.pos) {
		return s.Else.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	if isInRange(s.Condition.GetRange(), def.pos) {
		return s.Condition.Accept(def)
	}
	if isInRange(s.Body.GetRange(), def.pos) {
		return s.Body.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	if isInRange(s.Initializer.GetRange(), def.pos) {
		return s.Initializer.Accept(def)
	}
	if isInRange(s.To.GetRange(), def.pos) {
		return s.To.Accept(def)
	}
	if s.StepSize != nil && isInRange(s.StepSize.GetRange(), def.pos) {
		return s.StepSize.Accept(def)
	}
	if isInRange(s.Body.GetRange(), def.pos) {
		return s.Body.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	if isInRange(s.Initializer.GetRange(), def.pos) {
		return s.Initializer.Accept(def)
	}
	if isInRange(s.In.GetRange(), def.pos) {
		return s.In.Accept(def)
	}
	if isInRange(s.Body.GetRange(), def.pos) {
		return s.Body.Accept(def)
	}
	return def
}
func (def *definitionVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(def)
}
func (def *definitionVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value != nil && isInRange(s.Value.GetRange(), def.pos) {
		return s.Value.Accept(def)
	}
	return def
}