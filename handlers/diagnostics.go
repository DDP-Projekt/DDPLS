package handlers

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
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

type diagnosticParams struct {
	dm     *documents.DocumentManager
	notify glsp.NotifyFunc
	vscURI string
	delay  bool
}

func (d *diagnosticParams) sendDiagnostics() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic of type %s in diagnostics: %v", reflect.TypeOf(err), err)
			log.Errorf("stack trace: %s", string(debug.Stack()))
		}
	}()

	var (
		docMod *ast.Module
		docUri uri.URI
		errs   []ddperror.Error
	)
	if doc, ok := d.dm.Get(d.vscURI); !ok {
		log.Warningf("Could not retrieve for diagnostics document %s", d.vscURI)
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

	go d.notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         d.vscURI,
		Diagnostics: visitor.diagnostics,
	})
}

func CreateSendDiagnostics(ctx context.Context) DiagnosticSender {
	// initialize the timer stopped
	diagnosticsTimer := time.NewTimer(time.Nanosecond)
	diagnosticsTimer.Stop()

	done := ctx.Done()
	diagnosticChan := make(chan diagnosticParams)

	go func() {
		params := diagnosticParams{}
	infinite_loop:
		for {
			select {
			case new_params := <-diagnosticChan:
				if params.vscURI != new_params.vscURI {
					params.sendDiagnostics()
				}

				params = new_params
				if params.delay {
					diagnosticsTimer.Reset(time.Millisecond * 500)
				} else {
					params.sendDiagnostics()
				}
			case <-diagnosticsTimer.C:
				diagnosticsTimer.Stop()
				params.sendDiagnostics()
			case <-done:
				break infinite_loop
			}
		}
	}()

	return func(dm *documents.DocumentManager, notify glsp.NotifyFunc, vscURI string, delay bool) {
		diagnosticChan <- diagnosticParams{dm, notify, vscURI, delay}
	}
}

type diagnosticVisitor struct {
	path        string
	diagnostics []protocol.Diagnostic
}

var (
	_ ast.Visitor        = (*diagnosticVisitor)(nil)
	_ ast.BadDeclVisitor = (*diagnosticVisitor)(nil)
	_ ast.BadExprVisitor = (*diagnosticVisitor)(nil)
	_ ast.BadStmtVisitor = (*diagnosticVisitor)(nil)
)

func (d *diagnosticVisitor) add(err ddperror.Error) {
	severity := &severityError
	if err.Level == ddperror.LEVEL_WARN {
		severity = &severityWarning
	}

	diagnostic := protocol.Diagnostic{
		Range:    helper.ToProtocolRange(err.Range),
		Severity: severity,
		Source:   &errSrc,
		Message:  fmt.Sprintf("%s (%d)", err.Msg, err.Code),
		Code:     &protocol.IntegerOrString{Value: err.Code},
	}
	if err.File != d.path {
		diagnostic.Range = protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}}
		diagnostic.Message = err.File + ": " + diagnostic.Message
	}
	d.diagnostics = append(d.diagnostics, diagnostic)
}

var (
	severityError   = protocol.DiagnosticSeverityError
	severityWarning = protocol.DiagnosticSeverityWarning
	errSrc          = "ddp"
)

func (*diagnosticVisitor) Visitor() {}

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
