package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (interface{}, error) {
	doc, ok := documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
	}

	definition := &definitionVisitor{
		location: nil,
		pos:      params.Position,
	}

	ast.VisitModuleRec(doc.Module, definition)

	return definition.location, nil
}

type definitionVisitor struct {
	location *protocol.Location
	pos      protocol.Position
}

func (*definitionVisitor) BaseVisitor() {}

func (def *definitionVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), def.pos)
}

func (def *definitionVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {
		def.location = &protocol.Location{
			URI:   string(uri.FromPath(decl.Mod.FileName)),
			Range: helper.ToProtocolRange(decl.GetRange()),
		}
	}
}
func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) {
	if fun, ok := e.Func, e.Func != nil; ok {
		def.location = &protocol.Location{
			URI:   string(uri.FromPath(fun.Mod.FileName)),
			Range: helper.ToProtocolRange(fun.GetRange()),
		}
	}
}
