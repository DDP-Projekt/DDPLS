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
		definition := &definitionVisitor{
			location: nil,
			pos:      params.Position,
			dm:       dm,
		}

		if doc, ok := dm.Get(params.TextDocument.URI); !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		} else {
			definition.docMod = doc.Module
			definition.docUri = doc.Uri
		}

		ast.VisitModuleRec(definition.docMod, definition)

		return definition.location, nil
	}
}

type definitionVisitor struct {
	location *protocol.Location
	pos      protocol.Position
	dm       *documents.DocumentManager
	docMod   *ast.Module
	docUri   uri.URI
}

var _ ast.BaseVisitor = (*definitionVisitor)(nil)

func (*definitionVisitor) BaseVisitor() {}

func (def *definitionVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), def.pos)
}

func (def *definitionVisitor) VisitIdent(e *ast.Ident) ast.VisitResult {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(e.Declaration),
			Range: helper.ToProtocolRange(decl.GetRange()),
		}
	}
	return ast.VisitRecurse
}
func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	if fun, ok := e.Func, e.Func != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(fun),
			Range: helper.ToProtocolRange(fun.GetRange()),
		}
	}
	return ast.VisitRecurse
}
func (def *definitionVisitor) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	if struc, ok := e.Struct, e.Struct != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(struc),
			Range: helper.ToProtocolRange(struc.GetRange()),
		}
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) getUri(decl ast.Declaration) string {
	uri_ := uri.FromPath(decl.Module().FileName)
	if decl.Module() == def.docMod {
		uri_ = def.docUri
	} else if mod, ok := def.dm.GetFromMod(decl.Module()); ok {
		uri_ = mod.Uri
	}
	return string(uri_)
}
