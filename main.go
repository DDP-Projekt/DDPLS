package main

import (
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

var log = logging.GetLogger("ddp.ddpls")

func main() {
	// This increases logging verbosity (optional)
	logging.Configure(1, nil)

	handler = protocol.Handler{
		Initialize:                     initialize,
		Initialized:                    initialized,
		Shutdown:                       shutdown,
		SetTrace:                       setTrace,
		TextDocumentDidOpen:            textDocumentDidOpen,
		TextDocumentDidSave:            textDocumentDidSave,
		TextDocumentDidChange:          textDocumentDidChange,
		TextDocumentDidClose:           textDocumentDidClose,
		TextDocumentSemanticTokensFull: textDocumentSemanticTokensFull,
		//TextDocumentSemanticTokensRange: textDocumentSemanticTokensRange,
		//TextDocumentCompletion: textDocumentCompletion,
		//TextDocumentHover: textDocumentHover,
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
	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func tokenTypeLegend() []string {
	legend := make([]string, len(allTokenTypes))
	for i, tokenType := range allTokenTypes {
		legend[i] = string(tokenType)
	}
	return legend
}

func tokenModifierLegend() []string {
	legend := make([]string, len(allTokenModifiers))
	for i, tokenModifier := range allTokenModifiers {
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
