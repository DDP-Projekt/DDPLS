package handlers

import (
	"fmt"
	"sort"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/token"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentSemanticTokensFull(dm *documents.DocumentManager) protocol.TextDocumentSemanticTokensFullFunc {
	return func(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
		act, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("%s not in document map", params.TextDocument.URI)
		}
		path := act.Path

		tokenizer := &semanticTokenizer{
			tokens:          make([]highlightedToken, 0),
			file:            path,
			doc:             act,
			shouldVisitFunc: nil,
		}

		ast.VisitAst(act.Module.Ast, tokenizer)

		tokens := tokenizer.getTokens()
		return tokens, nil
	}
}
func CreateSemanticTokensRange(dm *documents.DocumentManager) protocol.TextDocumentSemanticTokensRangeFunc {
	return func(context *glsp.Context, params *protocol.SemanticTokensRangeParams) (any, error) {
		act, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("%s not in document map (Range: %v)", params.TextDocument.URI, params.Range)
		}
		path := act.Path

		tokenizer := &semanticTokenizer{
			tokens: make([]highlightedToken, 0),
			file:   path,
			doc:    act,
			shouldVisitFunc: func(node ast.Node) bool {
				rng := node.GetRange()
				return helper.IsInRange(rng, params.Range.Start) || helper.IsInRange(rng, params.Range.End)
			},
		}

		ast.VisitAst(act.Module.Ast, tokenizer)

		tokens := tokenizer.getTokens()
		return tokens, nil
	}
}

type highlightedToken struct {
	line, column, length int
	tokenType            protocol.SemanticTokenType
	modifiers            []protocol.SemanticTokenModifier
}

func (t *highlightedToken) serialize(previous highlightedToken) []protocol.UInteger {
	modifiers := protocol.UInteger(0)
	for _, modifier := range t.modifiers {
		modifiers |= bitFlag(modifier)
	}
	deltaLine := protocol.UInteger(t.line - previous.line)
	deltaStart := protocol.UInteger(0)
	if deltaLine == 0 {
		deltaStart = protocol.UInteger(t.column - previous.column)
	} else {
		deltaStart = protocol.UInteger(t.column - 1)
	}

	return []protocol.UInteger{
		deltaLine,
		deltaStart,
		protocol.UInteger(t.length),
		getTokenTypeIndex(t.tokenType),
		modifiers,
	}
}

type semanticTokenizer struct {
	file            string
	tokens          []highlightedToken
	doc             *documents.DocumentState
	shouldVisitFunc func(node ast.Node) bool
}

var _ ast.BaseVisitor = (*semanticTokenizer)(nil)

func (t *semanticTokenizer) ShouldVisit(node ast.Node) bool {
	if t.shouldVisitFunc != nil {
		return t.shouldVisitFunc(node)
	}
	return true
}

func (t *semanticTokenizer) getTokens() *protocol.SemanticTokens {
	data := make([]protocol.UInteger, 0, len(t.tokens)*5)
	for i := range t.tokens {
		if i == 0 {
			data = append(data, t.tokens[i].serialize(highlightedToken{line: 1, column: 1})...)
		} else {
			data = append(data, t.tokens[i].serialize(t.tokens[i-1])...)
		}
	}
	return &protocol.SemanticTokens{
		Data: data,
	}
}

func (*semanticTokenizer) BaseVisitor() {}

func (t *semanticTokenizer) add(tok highlightedToken) {
	t.tokens = append(t.tokens, tok)
}

func (t *semanticTokenizer) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	t.add(newHightlightedToken(token.NewRange(&d.NameTok, &d.NameTok), t.doc, protocol.SemanticTokenTypeVariable, nil))
	return ast.VisitRecurse
}
func (t *semanticTokenizer) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	t.add(newHightlightedToken(token.NewRange(&d.NameTok, &d.NameTok), t.doc, protocol.SemanticTokenTypeVariable, nil))
	for _, param := range d.ParamNames {
		t.add(newHightlightedToken(token.NewRange(&param, &param), t.doc, protocol.SemanticTokenTypeParameter, nil))
	}
	return ast.VisitRecurse
}
func (t *semanticTokenizer) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	/*for _, field := range d.Fields {
		switch field := field.(type) {
		case *ast.VarDecl:
			t.VisitVarDecl(field)
			t.VisitE
		}
	}*/
	t.add(newHightlightedToken(token.NewRange(&d.NameTok, &d.NameTok), t.doc, protocol.SemanticTokenTypeClass, nil))
	return ast.VisitRecurse
}

func (t *semanticTokenizer) VisitIdent(e *ast.Ident) ast.VisitResult {
	t.add(newHightlightedToken(e.GetRange(), t.doc, protocol.SemanticTokenTypeVariable, nil))
	return ast.VisitRecurse
}
func (t *semanticTokenizer) VisitIntLit(e *ast.IntLit) ast.VisitResult {
	t.add(newHightlightedToken(e.GetRange(), t.doc, protocol.SemanticTokenTypeNumber, nil))
	return ast.VisitRecurse
}
func (t *semanticTokenizer) VisitFloatLit(e *ast.FloatLit) ast.VisitResult {
	t.add(newHightlightedToken(e.GetRange(), t.doc, protocol.SemanticTokenTypeNumber, nil))
	return ast.VisitRecurse
}
func (t *semanticTokenizer) VisitCharLit(e *ast.CharLit) ast.VisitResult {
	t.add(newHightlightedToken(e.GetRange(), t.doc, protocol.SemanticTokenTypeString, nil))
	return ast.VisitRecurse
}
func (t *semanticTokenizer) VisitStringLit(e *ast.StringLit) ast.VisitResult {
	t.add(newHightlightedToken(e.GetRange(), t.doc, protocol.SemanticTokenTypeString, nil))
	return ast.VisitRecurse
}

func (t *semanticTokenizer) VisitFuncCall(e *ast.FuncCall) ast.VisitResult {
	rang := e.GetRange()
	if len(e.Args) != 0 {
		args := make([]ast.Expression, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, arg)
		}
		sort.Slice(args, func(i, j int) bool {
			iRange, jRange := args[i].GetRange(), args[j].GetRange()
			if iRange.Start.Line < jRange.Start.Line {
				return true
			}
			if iRange.Start.Line == jRange.Start.Line {
				return iRange.Start.Column < jRange.Start.Column
			}
			return false
		})

		for i, arg := range args {
			argRange := arg.GetRange()
			cutRange := helper.CutRangeOut(rang, argRange)
			if helper.GetRangeLength(cutRange[0], t.doc) != 0 {
				t.add(newHightlightedToken(cutRange[0], t.doc, protocol.SemanticTokenTypeFunction, nil))
			}
			ast.VisitNode(t, arg, nil)
			rang = token.Range{Start: cutRange[1].Start, End: rang.End}

			if i == len(e.Args)-1 && helper.GetRangeLength(cutRange[1], t.doc) != 0 {
				t.add(newHightlightedToken(cutRange[1], t.doc, protocol.SemanticTokenTypeFunction, nil))
			}
		}
	} else {
		t.add(newHightlightedToken(rang, t.doc, protocol.SemanticTokenTypeFunction, nil))
	}
	return ast.VisitRecurse
}

func (t *semanticTokenizer) VisitStructLiteral(e *ast.StructLiteral) ast.VisitResult {
	rang := e.GetRange()
	if len(e.Args) != 0 {
		args := make([]ast.Expression, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, arg)
		}
		sort.Slice(args, func(i, j int) bool {
			iRange, jRange := args[i].GetRange(), args[j].GetRange()
			if iRange.Start.Line < jRange.Start.Line {
				return true
			}
			if iRange.Start.Line == jRange.Start.Line {
				return iRange.Start.Column < jRange.Start.Column
			}
			return false
		})

		for i, arg := range args {
			argRange := arg.GetRange()
			cutRange := helper.CutRangeOut(rang, argRange)
			if helper.GetRangeLength(cutRange[0], t.doc) != 0 {
				t.add(newHightlightedToken(cutRange[0], t.doc, protocol.SemanticTokenTypeFunction, nil))
			}
			ast.VisitNode(t, arg, nil)
			rang = token.Range{Start: cutRange[1].Start, End: rang.End}

			if i == len(e.Args)-1 && helper.GetRangeLength(cutRange[1], t.doc) != 0 {
				t.add(newHightlightedToken(cutRange[1], t.doc, protocol.SemanticTokenTypeFunction, nil))
			}
		}
	} else {
		t.add(newHightlightedToken(rang, t.doc, protocol.SemanticTokenTypeFunction, nil))
	}
	return ast.VisitRecurse
}

func (t *semanticTokenizer) VisitImportStmt(e *ast.ImportStmt) ast.VisitResult {
	t.add(newHightlightedToken(e.FileName.Range, t.doc, protocol.SemanticTokenTypeString, nil))
	return ast.VisitRecurse
}

func newHightlightedToken(rang token.Range, doc *documents.DocumentState, tokType protocol.SemanticTokenType, modifiers []protocol.SemanticTokenModifier) highlightedToken {
	if modifiers == nil {
		modifiers = make([]protocol.SemanticTokenModifier, 0)
	}
	return highlightedToken{
		line:      int(rang.Start.Line),
		column:    int(rang.Start.Column),
		length:    helper.GetRangeLength(rang, doc),
		tokenType: tokType,
		modifiers: modifiers,
	}
}

// helper stuff for semantic tokens

var AllTokenTypes = []protocol.SemanticTokenType{
	protocol.SemanticTokenTypeNamespace,
	protocol.SemanticTokenTypeType,
	protocol.SemanticTokenTypeClass,
	protocol.SemanticTokenTypeEnum,
	protocol.SemanticTokenTypeInterface,
	protocol.SemanticTokenTypeStruct,
	protocol.SemanticTokenTypeTypeParameter,
	protocol.SemanticTokenTypeParameter,
	protocol.SemanticTokenTypeVariable,
	protocol.SemanticTokenTypeProperty,
	protocol.SemanticTokenTypeEnumMember,
	protocol.SemanticTokenTypeEvent,
	protocol.SemanticTokenTypeFunction,
	protocol.SemanticTokenTypeMethod,
	protocol.SemanticTokenTypeMacro,
	protocol.SemanticTokenTypeKeyword,
	protocol.SemanticTokenTypeModifier,
	protocol.SemanticTokenTypeComment,
	protocol.SemanticTokenTypeString,
	protocol.SemanticTokenTypeNumber,
	protocol.SemanticTokenTypeRegexp,
	protocol.SemanticTokenTypeOperator,
}

func getTokenTypeIndex(tokenType protocol.SemanticTokenType) protocol.UInteger {
	for i, t := range AllTokenTypes {
		if t == tokenType {
			return protocol.UInteger(i)
		}
	}
	return 0
}

var AllTokenModifiers = []protocol.SemanticTokenModifier{
	protocol.SemanticTokenModifierDeclaration,
	protocol.SemanticTokenModifierDefinition,
	protocol.SemanticTokenModifierReadonly,
	protocol.SemanticTokenModifierStatic,
	protocol.SemanticTokenModifierDeprecated,
	protocol.SemanticTokenModifierAbstract,
	protocol.SemanticTokenModifierAsync,
	protocol.SemanticTokenModifierModification,
	protocol.SemanticTokenModifierDocumentation,
	protocol.SemanticTokenModifierDefaultLibrary,
}

func bitFlag(modifier protocol.SemanticTokenModifier) protocol.UInteger {
	switch modifier {
	case protocol.SemanticTokenModifierDeclaration:
		return 0b0000000001
	case protocol.SemanticTokenModifierDefinition:
		return 0b0000000010
	case protocol.SemanticTokenModifierReadonly:
		return 0b0000000100
	case protocol.SemanticTokenModifierStatic:
		return 0b0000001000
	case protocol.SemanticTokenModifierDeprecated:
		return 0b0000010000
	case protocol.SemanticTokenModifierAbstract:
		return 0b0000100000
	case protocol.SemanticTokenModifierAsync:
		return 0b0001000000
	case protocol.SemanticTokenModifierModification:
		return 0b0010000000
	case protocol.SemanticTokenModifierDocumentation:
		return 0b0100000000
	case protocol.SemanticTokenModifierDefaultLibrary:
		return 0b1000000000
	}
	return 0
}
