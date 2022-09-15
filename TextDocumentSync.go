package main

import (
	"errors"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	documents.Add(params.TextDocument.URI, params.TextDocument.Text)
	documents.Active = params.TextDocument.URI
	sendDiagnostics(context.Notify, false)
	return nil
}

func textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	doc, ok := documents.Get(params.TextDocument.URI)
	if !ok {
		return errors.New("document sync error")
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
	documents.Active = params.TextDocument.URI
	sendDiagnostics(context.Notify, true)
	return nil
}

func textDocumentDidSave(*glsp.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	documents.Delete(params.TextDocument.URI)
	return nil
}
