package handlers

import (
	"fmt"
	"time"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var refreshing = false

func sendDiagnostics(notify glsp.NotifyFunc, docURI string, delay bool) {
	if refreshing {
		return
	}
	refreshing = true

	go func() {
		if delay {
			time.Sleep(500 * time.Millisecond)
		}
		refreshing = false

		act, ok := documents.Get(docURI)
		if !ok {
			log.Warningf("Could not retrieve document %s", docURI)
			return
		}
		path := act.Uri.Filepath()

		visitor := &diagnosticVisitor{path: path, diagnostics: make([]protocol.Diagnostic, 0)}

		if err := act.ReParse(func(err ddperror.Error) {
			visitor.add(err)
		}); err != nil {
			log.Errorf("parser error: %s", err)
			return
		}

		ast.VisitModuleRec(act.Module, visitor)

		go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         docURI,
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
