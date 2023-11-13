package handlers

import (
	"fmt"
	"time"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DiagnosticSender func(*documents.DocumentManager, glsp.NotifyFunc, string, bool)

func CreateSendDiagnostics() DiagnosticSender {
	refreshing := false
	return func(dm *documents.DocumentManager, notify glsp.NotifyFunc, vscURI string, delay bool) {
		if refreshing {
			return
		}
		refreshing = true

		go func(vscURI string) {
			if delay {
				time.Sleep(500 * time.Millisecond)
			}
			refreshing = false

			var (
				docMod *ast.Module
				docUri uri.URI
				errs   []ddperror.Error
			)
			if doc, ok := dm.Get(vscURI); !ok {
				log.Warningf("Could not retrieve document %s", vscURI)
				return
			} else {
				docMod = doc.Module
				docUri = doc.Uri
				errs = doc.LatestErrors
			}
			path := docUri.Filepath()

			visitor := &diagnosticVisitor{path: path, diagnostics: make([]protocol.Diagnostic, 0)}

			for i := range errs {
				visitor.add(errs[i])
			}

			ast.VisitModuleRec(docMod, visitor)

			go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
				URI:         vscURI,
				Diagnostics: visitor.diagnostics,
			})
		}(vscURI)
	}
}

type diagnosticVisitor struct {
	path        string
	diagnostics []protocol.Diagnostic
}

var _ ast.BaseVisitor = (*diagnosticVisitor)(nil)

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

func (d *diagnosticVisitor) VisitBadDecl(decl *ast.BadDecl) ast.VisitResult {
	if decl.Tok.Type != token.FUNKTION { // bad function declaration errors were already reported
		d.add(decl.Err)
	}
	return ast.VisitRecurse
}
func (d *diagnosticVisitor) VisitBadExpr(e *ast.BadExpr) ast.VisitResult {
	d.add(e.Err)
	return ast.VisitRecurse
}
func (d *diagnosticVisitor) VisitBadStmt(s *ast.BadStmt) ast.VisitResult {
	d.add(s.Err)
	return ast.VisitRecurse
}
