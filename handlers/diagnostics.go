package handlers

import (
	"time"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
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
		if currentAst, err = parse.WithErrorHandler(func(err ddperror.Error) {
			visitor.add(err, protocol.Diagnostic{
				Range:    helper.ToProtocolRange(err.GetRange()),
				Severity: &severityError,
				Source:   &errSrc,
				Message:  err.Msg(),
			})
		}); err != nil {
			log.Errorf("parser error: %s", err)
			return
		}

		ast.VisitAst(currentAst, visitor)

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

func (d *diagnosticVisitor) add(err ddperror.Error, diagnostic protocol.Diagnostic) {
	if err.File() != d.path {
		diagnostic.Range = protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}}
		diagnostic.Message = err.File() + ": " + diagnostic.Message
	}
	d.diagnostics = append(d.diagnostics, diagnostic)
}

var (
	severityError = protocol.DiagnosticSeverityError
	errSrc        = "ddp"
)

func (*diagnosticVisitor) BaseVisitor() {}

func (d *diagnosticVisitor) VisitBadDecl(decl *ast.BadDecl) {
	if decl.Tok.Type != token.FUNKTION { // bad function declaration errors were already reported
		d.add(decl.Err, protocol.Diagnostic{
			Range:    helper.ToProtocolRange(decl.GetRange()),
			Severity: &severityError,
			Source:   &errSrc,
			Message:  decl.Err.Msg(),
		})
	}
}
func (d *diagnosticVisitor) VisitBadExpr(e *ast.BadExpr) {
	d.add(e.Err, protocol.Diagnostic{
		Range:    helper.ToProtocolRange(e.GetRange()),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  e.Err.Msg(),
	})
}
func (d *diagnosticVisitor) VisitBadStmt(s *ast.BadStmt) {
	d.add(s.Err, protocol.Diagnostic{
		Range:    helper.ToProtocolRange(s.GetRange()),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  s.Err.Msg(),
	})
}
