package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	documents.Add(params.TextDocument.URI, params.TextDocument.Text)
	documents.Active = params.TextDocument.URI
	if _, err := parse.WithoutHandler(); err != nil {
		log.Warningf("Error while parsing new document %s: %s", documents.Active, err)
	}
	sendDiagnostics(context.Notify, false)
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
	documents.Active = params.TextDocument.URI
	if _, err := parse.WithoutHandler(); err != nil {
		log.Warningf("Error while parsing changed document %s: %s", documents.Active, err)
	}
	sendDiagnostics(context.Notify, true)
	return nil
}

func TextDocumentDidSave(*glsp.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	documents.Delete(params.TextDocument.URI)
	return nil
}
