package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentDidOpen(dm *documents.DocumentManager, sendDiagnostics DiagnosticSender) protocol.TextDocumentDidOpenFunc {
	return RecoverErr(func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		err := dm.AddAndParse(params.TextDocument.URI, params.TextDocument.Text)
		if err != nil {
			return fmt.Errorf("error while parsing module %s: %s", params.TextDocument.URI, err)
		}
		sendDiagnostics(dm, context.Notify, params.TextDocument.URI, false)
		return nil
	})
}

func CreateTextDocumentDidChange(dm *documents.DocumentManager, sendDiagnostics DiagnosticSender) protocol.TextDocumentDidChangeFunc {
	return RecoverErr(func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		doc, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return fmt.Errorf("%s not in document map", params.TextDocument.URI)
		}
		for _, change := range params.ContentChanges {
			switch change := change.(type) {
			case protocol.TextDocumentContentChangeEvent:
				startIndex, endIndex := change.Range.IndexesIn(doc.Content)
				doc.Content = doc.Content[:startIndex] + change.Text + doc.Content[endIndex:]
			case protocol.TextDocumentContentChangeEventWhole:
				doc.Content = change.Text
			}
			doc.NeedReparse.Store(true)
		}
		sendDiagnostics(dm, context.Notify, string(doc.Uri), true)
		return nil
	})
}

func TextDocumentDidSave(*glsp.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func CreateTextDocumentDidClose(dm *documents.DocumentManager) protocol.TextDocumentDidCloseFunc {
	return RecoverErr(func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		dm.Delete(params.TextDocument.URI)
		return nil
	})
}
