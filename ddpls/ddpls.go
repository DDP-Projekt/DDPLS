package ddpls

import (
	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/handlers"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	lspserver "github.com/tliron/glsp/server"

	// Must include a backend implementation. See kutil's logging/ for other options.
	_ "github.com/tliron/commonlog/simple"
)

const (
	lsName  = "ddp"
	version = "0.0.1"
)

type DDPLS struct {
	handler          protocol.Handler
	dm               *documents.DocumentManager
	diagnosticSender handlers.DiagnosticSender
	Server           *lspserver.Server
}

func NewDDPLS() *DDPLS {
	ls := &DDPLS{
		dm:               documents.NewDocumentManager(),
		diagnosticSender: handlers.CreateSendDiagnostics(),
	}
	ls.handler = protocol.Handler{
		Initialize:                      ls.createInitialize(),
		Initialized:                     initialized,
		Shutdown:                        shutdown,
		SetTrace:                        setTrace,
		TextDocumentDidOpen:             handlers.CreateTextDocumentDidOpen(ls.dm, ls.diagnosticSender),
		TextDocumentDidSave:             handlers.TextDocumentDidSave,
		TextDocumentDidChange:           handlers.CreateTextDocumentDidChange(ls.dm, ls.diagnosticSender),
		TextDocumentDidClose:            handlers.CreateTextDocumentDidClose(ls.dm),
		TextDocumentSemanticTokensFull:  handlers.CreateTextDocumentSemanticTokensFull(ls.dm),
		TextDocumentSemanticTokensRange: handlers.CreateSemanticTokensRange(ls.dm),
		TextDocumentCompletion:          handlers.CreateTextDocumentCompletion(ls.dm),
		TextDocumentHover:               handlers.CreateTextDocumentHover(ls.dm),
		TextDocumentDefinition:          handlers.CreateTextDocumentDefinition(ls.dm),
		TextDocumentFoldingRange:        handlers.CreateTextDocumentFoldingRange(ls.dm),
		TextDocumentRename:              handlers.CreateTextDocumentRename(ls.dm),
		TextDocumentPrepareRename:       handlers.CreateTextDocumentPrepareRename(ls.dm),
		TextDocumentDocumentHighlight:   handlers.CreateTextDocumentDocumentHighlight(ls.dm),
	}
	ls.Server = lspserver.NewServer(&ls.handler, lsName, false)
	return ls
}

func (ls *DDPLS) createInitialize() protocol.InitializeFunc {
	return handlers.RecoverAnyErr(func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		if params.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport != nil {
			handlers.SupportsSnippets = *params.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport
		}

		capabilities := ls.handler.CreateServerCapabilities()
		capabilities.SemanticTokensProvider = protocol.SemanticTokensRegistrationOptions{
			SemanticTokensOptions: protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenTypes:     tokenTypeLegend(),
					TokenModifiers: tokenModifierLegend(),
				},
				Full:  true,
				Range: true,
			},
		}
		capabilities.CompletionProvider = &protocol.CompletionOptions{
			TriggerCharacters: []string{
				"\"",
				"/",
				".",
			},
		}
		temp := true
		capabilities.RenameProvider = &protocol.RenameOptions{
			PrepareProvider: &temp,
		}
		capabilities.DocumentHighlightProvider = &protocol.DocumentHighlightOptions{
			WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{WorkDoneProgress: &temp},
		}
		version := version
		return protocol.InitializeResult{
			Capabilities: capabilities,
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name:    lsName,
				Version: &version,
			},
		}, nil
	})
}

// helper for semantic token
func tokenTypeLegend() []string {
	legend := make([]string, len(handlers.AllTokenTypes))
	for i, tokenType := range handlers.AllTokenTypes {
		legend[i] = string(tokenType)
	}
	return legend
}

// helper for semantic token
func tokenModifierLegend() []string {
	legend := make([]string, len(handlers.AllTokenModifiers))
	for i, tokenModifier := range handlers.AllTokenModifiers {
		legend[i] = string(tokenModifier)
	}
	return legend
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func shutdown(context *glsp.Context) error {
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}
