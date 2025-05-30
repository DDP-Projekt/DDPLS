package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentDefinition(dm *documents.DocumentManager) protocol.TextDocumentDefinitionFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
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
	})
}

type definitionVisitor struct {
	location *protocol.Location
	pos      protocol.Position
	dm       *documents.DocumentManager
	docMod   *ast.Module
	docUri   uri.URI
}

var (
	_ ast.Visitor                = (*definitionVisitor)(nil)
	_ ast.ConditionalVisitor     = (*definitionVisitor)(nil)
	_ ast.VarDeclVisitor         = (*definitionVisitor)(nil)
	_ ast.FuncDeclVisitor        = (*definitionVisitor)(nil)
	_ ast.TypeDefDeclVisitor     = (*definitionVisitor)(nil)
	_ ast.TypeAliasDeclVisitor   = (*definitionVisitor)(nil)
	_ ast.ImportStmtVisitor      = (*definitionVisitor)(nil)
	_ ast.CastExprVisitor        = (*definitionVisitor)(nil)
	_ ast.CastAssigneableVisitor = (*definitionVisitor)(nil)
)

func (*definitionVisitor) Visitor() {}

func (def *definitionVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), def.pos)
}

func (def *definitionVisitor) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	if helper.IsInRange(d.TypeRange, def.pos) {
		def.gotoType(d.Type)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	if helper.IsInRange(d.ReturnTypeRange, def.pos) {
		def.gotoType(d.ReturnType)
		return ast.VisitBreak
	}

	for _, param := range d.Parameters {
		if helper.IsInRange(param.TypeRange, def.pos) {
			def.gotoType(param.Type.Type)
			return ast.VisitBreak
		}
	}

	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	for _, field := range d.Fields {
		d, ok := field.(*ast.VarDecl)
		if !ok {
			continue
		}

		if helper.IsInRange(d.TypeRange, def.pos) {
			def.gotoType(d.Type)
			return ast.VisitBreak
		}
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitTypeAliasDecl(d *ast.TypeAliasDecl) ast.VisitResult {
	if helper.IsInRange(d.UnderlyingRange, def.pos) {
		def.gotoType(d.Underlying)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitTypeDefDecl(d *ast.TypeDefDecl) ast.VisitResult {
	if helper.IsInRange(d.UnderlyingRange, def.pos) {
		def.gotoType(d.Underlying)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitImportStmt(stmt *ast.ImportStmt) ast.VisitResult {
	if helper.IsInRange(stmt.FileName.Range, def.pos) {
		if stmt.SingleModule() == nil {
			return ast.VisitBreak
		}

		def.location = &protocol.Location{
			URI: protocol.DocumentUri(uri.FromPath(stmt.SingleModule().FileName)),
		}
		return ast.VisitBreak
	}

	for _, symbol := range stmt.ImportedSymbols {
		if helper.IsInRange(symbol.Range, def.pos) {
			if decl, ok, _ := def.docMod.Ast.Symbols.LookupDecl(symbol.Literal); ok {
				def.location = &protocol.Location{
					URI:   def.getUri(decl),
					Range: helper.ToProtocolRange(decl.GetRange()),
				}

				return ast.VisitBreak
			}
		}
	}

	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitIdent(e *ast.Ident) ast.VisitResult {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(e.Declaration),
			Range: helper.ToProtocolRange(decl.GetRange()),
		}
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), def.pos) {
				return ast.VisitRecurse
			}
		}
	}

	if fun, ok := e.Func, e.Func != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(fun),
			Range: helper.ToProtocolRange(fun.GetRange()),
		}
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), def.pos) {
				return ast.VisitRecurse
			}
		}
	}

	if struc, ok := e.Struct, e.Struct != nil; ok {
		def.location = &protocol.Location{
			URI:   def.getUri(struc),
			Range: helper.ToProtocolRange(struc.GetRange()),
		}
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitCastExpr(expr *ast.CastExpr) ast.VisitResult {
	if helper.IsInRange(expr.Range, def.pos) && !helper.IsInRange(expr.Lhs.GetRange(), def.pos) {
		def.gotoType(expr.TargetType)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) VisitCastAssigneable(expr *ast.CastAssigneable) ast.VisitResult {
	if helper.IsInRange(expr.Range, def.pos) && !helper.IsInRange(expr.Lhs.GetRange(), def.pos) {
		def.gotoType(expr.TargetType)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (def *definitionVisitor) getUri(decl ast.Declaration) string {
	if funDecl, ok := decl.(*ast.FuncDecl); ok && ast.IsGenericInstantiation(funDecl) {
		return def.getUri(funDecl.GenericInstantiation.GenericDecl)
	}

	uri_ := uri.FromPath(decl.Module().FileName)
	if decl.Module() == def.docMod {
		uri_ = def.docUri
	} else if mod, ok := def.dm.GetFromMod(decl.Module()); ok {
		uri_ = mod.Uri
	}
	return string(uri_)
}

func (def *definitionVisitor) gotoType(typ ddptypes.Type) {
	if typ == nil {
		return
	}

	if lt, ok := typ.(*ddptypes.ListType); ok {
		def.gotoType(lt.ElementType)
		return
	}

	name := typ.String()
	if structType, ok := typ.(*ddptypes.StructType); ok {
		name = structType.Name
	}

	decl, exists, _ := def.docMod.Ast.Symbols.LookupDecl(name)
	if !exists {
		return
	}

	uri := def.getUri(decl)
	Range := token.Range{}

	switch decl := decl.(type) {
	case *ast.StructDecl:
		Range = decl.NameTok.Range
	case *ast.TypeDefDecl:
		Range = decl.NameTok.Range
	case *ast.TypeAliasDecl:
		Range = decl.NameTok.Range
	}

	def.location = &protocol.Location{
		URI:   uri,
		Range: helper.ToProtocolRange(Range),
	}
}
