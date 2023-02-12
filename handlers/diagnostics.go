package handlers

import (
	"fmt"
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

	activeDoc := documents.Active // save it if it changes during the delay
	go func() {
		if delay {
			time.Sleep(500 * time.Millisecond)
		}
		refreshing = false

		act, ok := documents.Get(activeDoc)
		if !ok {
			log.Warningf("Could not retrieve document %s", activeDoc)
			return
		}
		path := act.Uri.Filepath()

		visitor := &diagnosticVisitor{path: path, diagnostics: make([]protocol.Diagnostic, 0)}

		var currentAst *ast.Ast
		var err error
		if currentAst, err = parse.WithErrorHandler(func(err ddperror.Error) {
			visitor.add(err)
		}); err != nil {
			log.Errorf("parser error: %s", err)
			return
		}

		ast.VisitAst(currentAst, visitor)

		go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         string(activeDoc),
			Diagnostics: visitor.diagnostics,
		})
	}()
}

type diagnosticVisitor struct {
	path        string
	diagnostics []protocol.Diagnostic
}

func (d *diagnosticVisitor) add(err ddperror.Error) {
	diagnostic := protocol.Diagnostic{
		Range:    helper.ToProtocolRange(err.Range),
		Severity: &severityError,
		Source:   &errSrc,
		Message:  fmt.Sprintf("%s (%d)", err.Msg, err.Code),
	}
	if err.File != d.path {
		diagnostic.Range = protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}}
		diagnostic.Message = err.File + ": " + diagnostic.Message
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
		d.add(decl.Err)
	}
}
func (d *diagnosticVisitor) VisitBadExpr(e *ast.BadExpr) {
	d.add(e.Err)
}
func (d *diagnosticVisitor) VisitBadStmt(s *ast.BadStmt) {
	d.add(s.Err)
}
