package handlers

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
	"slices"
	"time"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DiagnosticSender func(*documents.DocumentManager, glsp.NotifyFunc, string, bool)

type diagnosticParams struct {
	dm     *documents.DocumentManager
	notify glsp.NotifyFunc
	vscURI uri.URI
	delay  bool
}

type modImport struct {
	mod   *ast.Module
	imprt *ast.ImportStmt
}

func sendDiagnostics(params *diagnosticParams) {
	sendDiagnosticsRec(params, make(map[uri.URI]struct{}), nil, nil)
}

func sendDiagnosticsRec(params *diagnosticParams, alreadySent map[uri.URI]struct{}, mod *ast.Module, externalErrors []*ddperror.Error) {
	if _, ok := alreadySent[params.vscURI]; ok {
		return
	}
	alreadySent[params.vscURI] = struct{}{}

	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic of type %s in diagnostics: %v", reflect.TypeOf(err), err)
			log.Errorf("stack trace: %s", string(debug.Stack()))
		}
	}()

	var (
		docMod = mod
		docUri = params.vscURI
		errs   = externalErrors
	)
	if doc, ok := params.dm.Get(string(params.vscURI)); !ok && mod == nil {
		log.Warningf("Could not retrieve for diagnostics document %s (%s)", params.vscURI)
		return
	} else if ok {
		docMod = doc.Module
		docUri = doc.Uri
		errs = toPointerSlice(doc.LatestErrors)
	}

	path := docUri.Filepath()
	diagnostics := make([]protocol.Diagnostic, 0, len(errs))
	faultyImports := make(map[string][]*ddperror.Error, len(docMod.Imports))
	moduleMap := make(map[string]modImport, len(docMod.Imports))

	for _, err := range errs {
		if err.File == path {
			diagnostics = append(diagnostics, errToDiagnostic(err, path))
			continue
		}

		externalErrors := faultyImports[err.File]
		faultyImports[err.File] = append(externalErrors, err)

		if _, ok := moduleMap[err.File]; !ok {
			imprt := findModule(err.File, params.dm, docMod.Imports)
			if imprt.mod != nil {
				moduleMap[err.File] = imprt
			}
		}
	}

	for path, errs := range faultyImports {
		imprt := moduleMap[path]

		if imprt.imprt != nil {
			diagnostics = append(diagnostics, newImportDiagnostic(path, errs, imprt.imprt))
		}

		params := diagnosticParams{params.dm, params.notify, uri.FromPath(path), false}
		sendDiagnosticsRec(&params, alreadySent, imprt.mod, errs)

		continue
	}

	go params.notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         string(params.vscURI),
		Diagnostics: diagnostics,
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
				if params.vscURI != new_params.vscURI && params.vscURI != "" {
					sendDiagnostics(&params)
				}

				params = new_params
				if params.delay {
					diagnosticsTimer.Reset(time.Millisecond * 500)
				} else {
					sendDiagnostics(&params)
				}
			case <-diagnosticsTimer.C:
				diagnosticsTimer.Stop()
				sendDiagnostics(&params)
			case <-done:
				break infinite_loop
			}
		}
	}()

	return func(dm *documents.DocumentManager, notify glsp.NotifyFunc, vscURI string, delay bool) {
		diagnosticChan <- diagnosticParams{dm, notify, uri.FromURI(vscURI), delay}
	}
}

var (
	severityError   = protocol.DiagnosticSeverityError
	severityWarning = protocol.DiagnosticSeverityWarning
	errSrc          = "ddp"
)

func errToRelatedInformation(err *ddperror.Error) protocol.DiagnosticRelatedInformation {
	return protocol.DiagnosticRelatedInformation{
		Location: protocol.Location{
			URI:   protocol.DocumentUri(uri.FromPath(err.File)),
			Range: helper.ToProtocolRange(err.Range),
		},
		Message: err.Msg,
	}
}

func diagnosticRelatedInformationFromErr(err *ddperror.Error) []protocol.DiagnosticRelatedInformation {
	if len(err.WrappedGenericErrors) == 0 {
		return nil
	}

	result := make([]protocol.DiagnosticRelatedInformation, 0, len(err.WrappedGenericErrors))
	for _, err := range err.WrappedGenericErrors {
		result = append(result, errToRelatedInformation(&err))
		result = append(result, diagnosticRelatedInformationFromErr(&err)...)
	}
	return result
}

func errToDiagnostic(err *ddperror.Error, docPath string) protocol.Diagnostic {
	severity := &severityError
	if err.Level == ddperror.LEVEL_WARN {
		severity = &severityWarning
	}

	return protocol.Diagnostic{
		Range:              helper.ToProtocolRange(err.Range),
		Severity:           severity,
		Source:             &errSrc,
		Message:            fmt.Sprintf("%s (%d)", err.Msg, err.Code),
		Code:               &protocol.IntegerOrString{Value: err.Code},
		RelatedInformation: diagnosticRelatedInformationFromErr(err),
	}
}

func newImportDiagnostic(path string, errs []*ddperror.Error, imprt *ast.ImportStmt) protocol.Diagnostic {
	related := make([]protocol.DiagnosticRelatedInformation, 0, len(errs))

	for _, err := range errs {
		related = append(related, errToRelatedInformation(err))
	}

	return protocol.Diagnostic{
		Range:              helper.ToProtocolRange(imprt.GetRange()),
		Severity:           &severityError,
		Source:             &errSrc,
		Message:            fmt.Sprintf("Das eingebundene Modul '%s' enthält Fehler", path),
		Code:               &protocol.IntegerOrString{Value: ddperror.MISC_INCLUDE_ERROR},
		RelatedInformation: related,
	}
}

func findModule(path string, dm *documents.DocumentManager, imports []*ast.ImportStmt) modImport {
	// if doc, ok := dm.Get(string(uri.FromPath(path))); ok {
	// 	return doc.Module
	// }

	for _, imprt := range imports {
		index := slices.IndexFunc(imprt.Modules, func(mod *ast.Module) bool {
			return mod.FileName == path
		})

		if index != -1 {
			return modImport{imprt.Modules[index], imprt}
		}

		for _, mod := range imprt.Modules {
			if imprt := findModule(path, dm, mod.Imports); imprt.mod != nil {
				return imprt
			}
		}
	}

	return modImport{}
}

func toPointerSlice[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}
