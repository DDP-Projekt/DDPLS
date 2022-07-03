package main

import (
	"time"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var refreshing = false

func sendDiagnostics(notify glsp.NotifyFunc, delay bool) {
	if refreshing {
		return
	}
	refreshing = true

	if err := parse(); err != nil {
		log.Critical(err.Error())
		return
	}

	go func() {
		if delay {
			time.Sleep(500 * time.Millisecond)
		}
		refreshing = false

		visitor := &diagnosticVisitor{diagnostics: make([]protocol.Diagnostic, 0)}
		ast.WalkAst(currentAst, visitor)

		go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         activeDocument,
			Diagnostics: visitor.diagnostics,
		})
	}()
}

type diagnosticVisitor struct {
	diagnostics []protocol.Diagnostic
}

func (d *diagnosticVisitor) add(diagnostic protocol.Diagnostic) {
	d.diagnostics = append(d.diagnostics, diagnostic)
}

var severityError = protocol.DiagnosticSeverityError
var errSrc = "ddp"

func (d *diagnosticVisitor) VisitBadDecl(decl *ast.BadDecl) ast.Visitor {
	d.add(protocol.Diagnostic{
		Range:    toProtocolRange(decl.GetRange()),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  decl.Message,
	})
	return d
}
func (d *diagnosticVisitor) VisitVarDecl(decl *ast.VarDecl) ast.Visitor {
	return decl.InitVal.Accept(d)
}
func (d *diagnosticVisitor) VisitFuncDecl(decl *ast.FuncDecl) ast.Visitor {
	return decl.Body.Accept(d)
}

func (d *diagnosticVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	d.add(protocol.Diagnostic{
		Range:    toProtocolRange(e.GetRange()),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  e.Message,
	})
	return d
}
func (d *diagnosticVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitCharLit(e *ast.CharLit) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitStringLit(e *ast.StringLit) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	return e.Rhs.Accept(d)
}
func (d *diagnosticVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	e.Lhs.Accept(d)
	return e.Rhs.Accept(d)
}
func (d *diagnosticVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	e.Lhs.Accept(d)
	e.Mid.Accept(d)
	return e.Rhs.Accept(d)
}
func (d *diagnosticVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(d)
}
func (d *diagnosticVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	for _, arg := range e.Args {
		arg.Accept(d)
	}
	return d
}

func (d *diagnosticVisitor) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	d.add(protocol.Diagnostic{
		Range:    toProtocolRange(s.GetRange()),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  s.Message,
	})
	return d
}
func (d *diagnosticVisitor) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(d)
}
func (d *diagnosticVisitor) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(d)
}
func (d *diagnosticVisitor) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	return s.Rhs.Accept(d)
}
func (d *diagnosticVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	for _, stmt := range s.Statements {
		stmt.Accept(d)
	}
	return d
}
func (d *diagnosticVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	s.Condition.Accept(d)
	if s.Then != nil {
		s.Then.Accept(d)
	}
	if s.Else != nil {
		s.Else.Accept(d)
	}
	return d
}
func (d *diagnosticVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	s.Condition.Accept(d)
	return s.Body.Accept(d)
}
func (d *diagnosticVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	s.Initializer.Accept(d)
	s.To.Accept(d)
	if s.StepSize != nil {
		s.StepSize.Accept(d)
	}
	return s.Body.Accept(d)
}
func (d *diagnosticVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	s.Initializer.Accept(d)
	s.In.Accept(d)
	return s.Body.Accept(d)
}
func (d *diagnosticVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(d)
}
func (d *diagnosticVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	return s.Value.Accept(d)
}

func toProtocolRange(rang token.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(rang.Start.Line - 1),
			Character: uint32(rang.Start.Column),
		},
		End: protocol.Position{
			Line:      uint32(rang.End.Line - 1),
			Character: uint32(rang.End.Column),
		},
	}
}
