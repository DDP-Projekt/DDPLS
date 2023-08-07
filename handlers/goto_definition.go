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

func CreateTextDocumentDefinition(dm *documents.DocumentManager) protocol.TextDocumentDefinitionFunc {
	return func(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
		doc, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		}

		definition := &definitionVisitor{
			location: nil,
			pos:      params.Position,
			dm:       dm,
			doc:      doc,
		}

		ast.VisitModuleRec(doc.Module, definition)

		return definition.location, nil
	}
}

type definitionVisitor struct {
	location *protocol.Location
	pos      protocol.Position
	dm       *documents.DocumentManager
	doc      *documents.DocumentState
}

func (*definitionVisitor) BaseVisitor() {}

func (def *definitionVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), def.pos)
}

func (def *definitionVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {

		def.location = &protocol.Location{
			URI:   def.getUri(e.Declaration),
			Range: helper.ToProtocolRange(decl.GetRange()),
		}
	}
}
func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) {
	if fun, ok := e.Func, e.Func != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(e.Func),
			Range: helper.ToProtocolRange(fun.GetRange()),
		}
	}
}

func (def *definitionVisitor) getUri(decl ast.Declaration) string {
	uri_ := uri.FromPath(decl.Module().FileName)
	if decl.Module() == def.doc.Module {
		uri_ = def.doc.Uri
	} else if mod, ok := def.dm.GetFromMod(decl.Module()); ok {
		uri_ = mod.Uri
	}
	return string(uri_)
}
