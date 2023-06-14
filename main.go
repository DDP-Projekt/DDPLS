package main

import (
	"github.com/DDP-Projekt/DDPLS/handlers"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
	"github.com/tliron/kutil/logging"

	// Must include a backend implementation. See kutil's logging/ for other options.
	_ "github.com/tliron/kutil/logging/simple"
)

const lsName = "ddp"

var version string = "0.0.1"
var handler protocol.Handler

func main() {
	// This increases logging verbosity (optional)
	logging.Configure(1, nil)

	handler = protocol.Handler{
		Initialize:                     initialize,
		Initialized:                    initialized,
		Shutdown:                       shutdown,
		SetTrace:                       setTrace,
		TextDocumentDidOpen:            handlers.TextDocumentDidOpen,
		TextDocumentDidSave:            handlers.TextDocumentDidSave,
		TextDocumentDidChange:          handlers.TextDocumentDidChange,
		TextDocumentDidClose:           handlers.TextDocumentDidClose,
		TextDocumentSemanticTokensFull: handlers.TextDocumentSemanticTokensFull,
		TextDocumentCompletion:         handlers.TextDocumentCompletion,
		TextDocumentHover:              handlers.TextDocumentHover,
		TextDocumentDefinition:         handlers.TextDocumentDefinition,
		TextDocumentFoldingRange:       handlers.TextDocumentFoldingRange,
	}
	server := server.NewServer(&handler, lsName, false)

	server.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()
	capabilities.SemanticTokensProvider = protocol.SemanticTokensRegistrationOptions{
		SemanticTokensOptions: protocol.SemanticTokensOptions{
			Legend: protocol.SemanticTokensLegend{
				TokenTypes:     tokenTypeLegend(),
				TokenModifiers: tokenModifierLegend(),
			},
			Full: true,
		},
	}
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{
			"\"",
			"/",
		},
	}
	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
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
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}
