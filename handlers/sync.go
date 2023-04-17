package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	err := documents.AddAndParse(params.TextDocument.URI, params.TextDocument.Text)
	if err != nil {
		return fmt.Errorf("error while parsing module %s: %s", params.TextDocument.URI, err)
	}
	sendDiagnostics(context.Notify, params.TextDocument.URI, false)
	return nil
}

func TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	doc, ok := documents.Get(params.TextDocument.URI)
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
	}
	if err := doc.ReParse(ddperror.EmptyHandler); err != nil {
		log.Warningf("Error while parsing changed document %s: %s", doc.Path, err)
	}
	sendDiagnostics(context.Notify, string(doc.Uri), true)
	return nil
}

func TextDocumentDidSave(*glsp.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	documents.Delete(params.TextDocument.URI)
	return nil
}
