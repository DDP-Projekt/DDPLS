package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentHover(dm *documents.DocumentManager) protocol.TextDocumentHoverFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
		doc, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("%s not in document map", params.TextDocument.URI)
		}

		hover := &hoverVisitor{
			hover:          nil,
			pos:            params.Position,
			dm:             dm,
			file:           doc.Module.FileName,
			currentSymbols: doc.Module.Ast.Symbols,
			docContent:     doc.Content,
		}

		ast.VisitModule(doc.Module, hover)

		return hover.hover, nil
	})
}

const commentCutset = " \r\n\t[]"

func trimComment(comment *token.Token) string {
	if comment != nil {
		return strings.Trim(comment.Literal, commentCutset)
	}
	return ""
}

type hoverVisitor struct {
	hover          *protocol.Hover
	pos            protocol.Position
	currentSymbols ast.SymbolTable
	docContent     string
	file           string
	dm             *documents.DocumentManager
	vis            ast.FullVisitor
}

var (
	_ ast.Visitor              = (*hoverVisitor)(nil)
	_ ast.ConditionalVisitor   = (*hoverVisitor)(nil)
	_ ast.ScopeSetter          = (*hoverVisitor)(nil)
	_ ast.ImportStmtVisitor    = (*hoverVisitor)(nil)
	_ ast.VarDeclVisitor       = (*hoverVisitor)(nil)
	_ ast.FuncDeclVisitor      = (*hoverVisitor)(nil)
	_ ast.TypeDefDeclVisitor   = (*hoverVisitor)(nil)
	_ ast.TypeAliasDeclVisitor = (*hoverVisitor)(nil)
)

func (*hoverVisitor) Visitor() {}

func (h *hoverVisitor) ShouldVisit(node ast.Node) bool {
	return helper.IsInRange(node.GetRange(), h.pos)
}

func (h *hoverVisitor) SetVisitor(vis ast.FullVisitor) {
	h.vis = vis
}

func (h *hoverVisitor) SetScope(symbols ast.SymbolTable) {
	h.currentSymbols = symbols
}

func (h *hoverVisitor) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	if helper.IsInRange(d.TypeRange, h.pos) {
		h.typeHover(d.TypeRange, d.Type)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (h *hoverVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	if helper.IsInRange(d.ReturnTypeRange, h.pos) {
		h.typeHover(d.ReturnTypeRange, d.ReturnType)
		return ast.VisitBreak
	}

	for _, param := range d.Parameters {
		if helper.IsInRange(param.TypeRange, h.pos) {
			h.typeHover(param.TypeRange, param.Type.Type)
			return ast.VisitBreak
		}
	}

	if instantiation := getRandomGenericInstantiation(d); instantiation != nil {
		h.vis.VisitFuncDecl(instantiation)
		return ast.VisitSkipChildren
	}

	return ast.VisitRecurse
}

func (h *hoverVisitor) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	for _, field := range d.Fields {
		d, ok := field.(*ast.VarDecl)
		if !ok {
			continue
		}

		if helper.IsInRange(d.TypeRange, h.pos) {
			h.typeHover(d.TypeRange, d.Type)
			return ast.VisitBreak
		}
	}
	return ast.VisitRecurse
}

func (h *hoverVisitor) VisitTypeAliasDecl(d *ast.TypeAliasDecl) ast.VisitResult {
	if helper.IsInRange(d.UnderlyingRange, h.pos) {
		h.typeHover(d.UnderlyingRange, d.Underlying)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (h *hoverVisitor) VisitTypeDefDecl(d *ast.TypeDefDecl) ast.VisitResult {
	if helper.IsInRange(d.UnderlyingRange, h.pos) {
		h.typeHover(d.UnderlyingRange, d.Underlying)
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (h *hoverVisitor) VisitIdent(e *ast.Ident) ast.VisitResult {
	if decl, ok := e.Declaration, e.Declaration != nil; ok {
		header := ""
		if decl.Module().FileName != h.file {
			header = fmt.Sprintf("%s\n", h.getHoverFilePath(decl.Module().FileName))
		}
		comment := trimComment(decl.Comment())
		pRange := helper.ToProtocolRange(e.GetRange())

		var typ ddptypes.Type
		switch decl := decl.(type) {
		case *ast.ConstDecl:
			typ = decl.Type
		case *ast.VarDecl:
			typ = decl.Type
		}

		h.hover = &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf(
					"%s%s\n```ddp\n%s\n```", header, comment, typ,
				),
			},
			Range: &pRange,
		}
	}
	return ast.VisitBreak
}

func (h *hoverVisitor) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return ast.VisitRecurse
			}
		}
	}

	if e.Func == nil {
		return ast.VisitBreak
	}

	// for extern functions we display the whole function,
	// for normal functions only the first line until the colon
	var declRange protocol.Range
	if e.Func.Body != nil {
		declRange = helper.ToProtocolRange(token.NewRange(&e.Func.Tok, &e.Func.Body.Colon))
	} else {
		declRange = helper.ToProtocolRange(e.Func.GetRange())
	}

	genericMod := e.Func.Mod
	if ast.IsGenericInstantiation(e.Func) {
		genericMod = e.Func.GenericInstantiation.GenericDecl.Mod
	}

	moduleContent, is_same_module := h.getDifferentModContent(genericMod)

	// if the function is in another module, we display the path to that module
	header := ""
	if !is_same_module {
		header = h.getHoverFilePath(genericMod.FileName)
	}

	if ast.IsGenericInstantiation(e.Func) {
		header += "\n\n"
		for name, typ := range e.Func.GenericInstantiation.Types {
			header += name + " = " + typ.String() + ", "
		}
		header = header[:len(header)-2]
	}

	start, end := declRange.IndexesIn(moduleContent)
	body := moduleContent[start:end]
	if e.Func.Body != nil {
		body += "\n..."
		endRange := helper.ToProtocolRange(token.Range{
			Start: e.Func.Body.Range.End,
			End:   e.Func.GetRange().End,
		})
		start, end = endRange.IndexesIn(moduleContent)
		body += moduleContent[start:end]
	}

	comment := getCommentDisplayString(e.Func.Comment())

	pRange := helper.ToProtocolRange(e.GetRange())
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("%s\n%s```ddp\n%s\n```", header, comment, body),
		},
		Range: &pRange,
	}

	return ast.VisitBreak
}

func (h *hoverVisitor) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), h.pos) {
				return ast.VisitRecurse
			}
		}
	}

	if e.Struct == nil {
		return ast.VisitBreak
	}

	moduleContent, is_same_module := h.getDifferentModContent(e.Struct.Mod)

	header := ""
	if !is_same_module {
		header = h.getHoverFilePath(e.Struct.Mod.FileName) + "\n"
	}

	declRange := helper.ToProtocolRange(e.Struct.GetRange())
	start, end := declRange.IndexesIn(moduleContent)
	body := moduleContent[start:end]

	comment := getCommentDisplayString(e.Struct.Comment())
	pRange := helper.ToProtocolRange(e.GetRange())
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf(
				"%s%s\n```ddp\n%s\n```", header, comment, body,
			),
		},
		Range: &pRange,
	}
	return ast.VisitBreak
}

// TODO: list all public decls
func (h *hoverVisitor) VisitImportStmt(stmt *ast.ImportStmt) ast.VisitResult {
	if len(stmt.Modules) == 0 {
		return ast.VisitBreak
	}

	if len(stmt.Modules) > 1 {
		// TODO: implement multi module import
		return ast.VisitBreak
	}

	comment := getCommentDisplayString(stmt.SingleModule().Comment)

	variableSection := strings.Builder{}
	functionSection := strings.Builder{}
	structSection := strings.Builder{}

	for _, decl := range stmt.SingleModule().PublicDecls {
		switch decl := decl.(type) {
		case *ast.VarDecl:
			switch decl.Type.Gender() {
			case ddptypes.MASKULIN:
				variableSection.WriteString("Den ")
			case ddptypes.FEMININ:
				variableSection.WriteString("Die ")
			case ddptypes.NEUTRUM:
				variableSection.WriteString("Das ")
			}

			variableSection.WriteString(decl.Type.String() + " " + decl.Name() + ".\n")
		case *ast.FuncDecl:
			functionSection.WriteString(fmt.Sprintf("Die Funktion %s.\n", decl.Name()))
		case *ast.StructDecl:
			structSection.WriteString(fmt.Sprintf("Die Kombination %s.\n", decl.Name()))
		}
	}

	result := fmt.Sprintf(
		"%s\n\n%s deklariert:\n\n```ddp\n%s\n%s\n%s\n```\n",
		comment,
		h.getHoverFilePath(stmt.SingleModule().FileName),
		structSection.String(),
		variableSection.String(),
		functionSection.String(),
	)

	pRange := helper.ToProtocolRange(stmt.GetRange())
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: result,
		},
		Range: &pRange,
	}
	return ast.VisitBreak
}

func (h *hoverVisitor) getDifferentModContent(mod *ast.Module) (string, bool) {
	is_same_module := mod.FileName == h.file
	// retreive the content of the file in which the function is defined
	moduleContent := h.docContent
	if doc, ok := h.dm.GetFromMod(mod); !is_same_module && ok { // if we already read the file, reuse it
		moduleContent = doc.Content
	} else if !is_same_module { // read the new file
		if content, err := os.ReadFile(mod.FileName); err != nil {
			log.Errorf("Unable to read %s: %s", mod.FileName, err)
		} else {
			moduleContent = string(content)
		}
	}
	return moduleContent, is_same_module
}

// helper to get a nice-looking path to display in a hover
func (h *hoverVisitor) getHoverFilePath(file string) string {
	datei, err := filepath.Rel(h.file, file)
	if err != nil {
		datei = filepath.Base(file)
	} else {
		datei = filepath.ToSlash(strings.TrimPrefix(datei, ".."+string(filepath.Separator)))
	}
	if strings.HasPrefix(file, ddppath.Duden) {
		datei = "Duden/" + filepath.Base(file)
	}
	return datei
}

// TODO: make comments prettier
func getCommentDisplayString(comment *token.Token) string {
	if comment == nil {
		return ""
	}

	result := trimComment(comment)
	if strings.Contains(result, "\n") {
		return fmt.Sprintf("```ddp\n[\n%s\n]\n```\n", result)
	} else {
		return fmt.Sprintf("```ddp\n[%s]\n```\n", result)
	}
}

func (h *hoverVisitor) typeHover(rang token.Range, typ ddptypes.Type) {
	if typ == nil {
		return
	}

	value := typ.String()
	switch typ := typ.(type) {
	case *ddptypes.StructType:
		value += ":\n"
		for _, field := range typ.Fields {
			value += fmt.Sprintf("\t%s: %s\n", field.Name, field.Type.String())
		}
	case *ddptypes.TypeAlias:
		value += fmt.Sprintf(" (Alias für %s)", typ.Underlying.String())
	case *ddptypes.TypeDef:
		value += fmt.Sprintf(" (definiert als %s)", typ.Underlying.String())
	}

	pRang := helper.ToProtocolRange(rang)
	h.hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: value,
		},
		Range: &pRang,
	}
}
