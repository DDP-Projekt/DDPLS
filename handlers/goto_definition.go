package handlers

import (
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (interface{}, error) {
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.ReparseIfNotActive(params.TextDocument.URI); err != nil {
		return nil, err
	}

	definition := &definitionVisitor{
		location:       nil,
		currentSymbols: nil,
		currentAst:     currentAst,
		pos:            params.Position,
	}

	ast.VisitAst(currentAst, definition)

	return definition.location, nil
}

type definitionVisitor struct {
	location       *protocol.Location
	currentSymbols *ast.SymbolTable
	currentAst     *ast.Ast
	pos            protocol.Position
}

func (*definitionVisitor) BaseVisitor() {}

func (def *definitionVisitor) ShouldVisit(node ast.Node) bool {
	return node.Token().File == def.currentAst.File && helper.IsInRange(node.GetRange(), def.pos)
}

func (def *definitionVisitor) UpdateScope(symbols *ast.SymbolTable) {
	def.currentSymbols = symbols
}

func (def *definitionVisitor) VisitIdent(e *ast.Ident) {
	if decl, ok := def.currentSymbols.LookupVar(e.Literal.Literal); ok {
		def.location = &protocol.Location{
			URI:   string(uri.FromPath(decl.Token().File)),
			Range: helper.ToProtocolRange(decl.GetRange()),
		}
	}
}
func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) {
	if fun, ok := def.currentSymbols.LookupFunc(e.Name); ok {
		def.location = &protocol.Location{
			URI:   string(uri.FromPath(fun.Token().File)),
			Range: helper.ToProtocolRange(fun.GetRange()),
		}
	}
}
