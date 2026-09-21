package handlers

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/DDPLS/documents"
	formatierer "github.com/DDP-Projekt/Formatierer"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentFormatting(dm *documents.DocumentManager) protocol.TextDocumentFormattingFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
		doc, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		}

		if len(doc.Content) == 0 {
			return make([]protocol.TextEdit, 0), nil
		}

		docRange := fullDocumentRange(doc.Content)

		spaces := false
		if insertSpaces, exists := params.Options["insertSpaces"]; exists && insertSpaces == true {
			spaces = true
		}

		opts := formatierer.FormattingOptions{
			InsertSpaces: spaces,
		}
		newText := formatierer.GetFormattedDocument(doc.Content, *doc.Module, opts)

		return []protocol.TextEdit{
			{
				Range:   docRange,
				NewText: newText,
			},
		}, nil
	})
}

func fullDocumentRange(content string) protocol.Range {
	lines := strings.Split(content, "\n")
	lastLine := lines[len(lines)-1]

	return protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End: protocol.Position{
			Line:      uint32(len(lines) - 1),
			Character: uint32(utf8.RuneCountInString(lastLine)),
		},
	}
}
