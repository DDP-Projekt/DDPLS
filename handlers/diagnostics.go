package handlers

import (
	"time"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/DDPLS/parse"
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

	go func() {
		if delay {
			time.Sleep(500 * time.Millisecond)
		}
		refreshing = false

		act, _ := documents.Get(documents.Active)
		path := act.Uri.Filepath()

		visitor := &diagnosticVisitor{path: path, diagnostics: make([]protocol.Diagnostic, 0)}

		var currentAst *ast.Ast
		var err error
		if currentAst, err = parse.WithErrorHandler(func(tok token.Token, msg string) {
			visitor.add(tok, protocol.Diagnostic{
				Range:    helper.ToProtocolRange(token.NewRange(tok, tok)),
				Severity: &severityError,
				Source:   &errSrc,
				Message:  msg,
			})
		}); err != nil {
			log.Errorf("parser error: %s", err)
			return
		}

		for _, stmt := range currentAst.Statements {
			stmt.Accept(visitor)
		}

		go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         string(documents.Active),
			Diagnostics: visitor.diagnostics,
		})
	}()
}

type diagnosticVisitor struct {
	path        string
	diagnostics []protocol.Diagnostic
}

func (d *diagnosticVisitor) add(tok token.Token, diagnostic protocol.Diagnostic) {
	if tok.File != d.path {
		diagnostic.Range = protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}}
		diagnostic.Message = tok.File + ": " + diagnostic.Message
	}
	d.diagnostics = append(d.diagnostics, diagnostic)
}

var (
	severityError = protocol.DiagnosticSeverityError
	errSrc        = "ddp"
)

func (d *diagnosticVisitor) VisitBadDecl(decl *ast.BadDecl) ast.Visitor {
	if decl.Tok.Type != token.FUNKTION { // bad function declaration errors were already reported
		d.add(decl.Token(), protocol.Diagnostic{
			Range:    helper.ToProtocolRange(decl.GetRange()),
			Severity: &severityError,
			Source:   &errSrc,
			Message:  decl.Message,
		})
	}
	return d
}
func (d *diagnosticVisitor) VisitVarDecl(decl *ast.VarDecl) ast.Visitor {
	return decl.InitVal.Accept(d)
}
func (d *diagnosticVisitor) VisitFuncDecl(decl *ast.FuncDecl) ast.Visitor {
	if decl.Body != nil {
		decl.Body.Accept(d)
	}
	return d
}

func (d *diagnosticVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	d.add(e.Token(), protocol.Diagnostic{
		Range:    helper.ToProtocolRange(e.GetRange()),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  e.Message,
	})
	return d
}
func (d *diagnosticVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitIndexing(e *ast.Indexing) ast.Visitor {
	e.Lhs.Accept(d)
	return e.Index.Accept(d)
}
func (d *diagnosticVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return d
}
func (d *diagnosticVisitor) VisitFloatLit(e *ast.FloatLit) ast.Visitor {
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
func (d *diagnosticVisitor) VisitListLit(e *ast.ListLit) ast.Visitor {
	if e.Values != nil {
		for _, expr := range e.Values {
			expr.Accept(d)
		}
	} else if e.Count != nil && e.Value != nil {
		e.Count.Accept(d)
		e.Value.Accept(d)
	}
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
func (d *diagnosticVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	return e.Lhs.Accept(d)
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
	d.add(s.Token(), protocol.Diagnostic{
		Range:    helper.ToProtocolRange(s.GetRange()),
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
	s.Var.Accept(d)
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
	switch s.While.Type {
	case token.SOLANGE:
		s.Condition.Accept(d)
		s.Body.Accept(d)
	case token.MACHE, token.COUNT_MAL:
		s.Body.Accept(d)
		s.Condition.Accept(d)
	}
	return d
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
	if s.Value == nil {
		return d
	}
	return s.Value.Accept(d)
}
