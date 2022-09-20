package handlers

import (
	"sort"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentSemanticTokensFull(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}

	tokenizer := &semanticTokenizer{}

	act, _ := documents.Get(documents.Active)
	path := act.Uri.Filepath()

	for _, stmt := range currentAst.Statements {
		if stmt.Token().File == path {
			stmt.Accept(tokenizer)
		}
	}

	return tokenizer.getTokens(), nil
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
	tokens []highlightedToken
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

func (t *semanticTokenizer) VisitBadDecl(d *ast.BadDecl) {
}
func (t *semanticTokenizer) VisitVarDecl(d *ast.VarDecl) {
	t.add(newHightlightedToken(token.NewRange(d.Name, d.Name), protocol.SemanticTokenTypeVariable, nil))
	d.InitVal.Accept(t)
}
func (t *semanticTokenizer) VisitFuncDecl(d *ast.FuncDecl) {
	t.add(newHightlightedToken(token.NewRange(d.Name, d.Name), protocol.SemanticTokenTypeVariable, nil))
	for _, param := range d.ParamNames {
		t.add(newHightlightedToken(token.NewRange(param, param), protocol.SemanticTokenTypeParameter, nil))
	}
	if d.Body != nil {
		d.Body.Accept(t)
	}
}

func (t *semanticTokenizer) VisitBadExpr(e *ast.BadExpr) {
}
func (t *semanticTokenizer) VisitIdent(e *ast.Ident) {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeVariable, nil))
}
func (t *semanticTokenizer) VisitIndexing(e *ast.Indexing) {
	e.Lhs.Accept(t)
	e.Index.Accept(t)
}
func (t *semanticTokenizer) VisitIntLit(e *ast.IntLit) {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeNumber, nil))
}
func (t *semanticTokenizer) VisitFloatLit(e *ast.FloatLit) {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeNumber, nil))
}
func (t *semanticTokenizer) VisitBoolLit(e *ast.BoolLit) {
}
func (t *semanticTokenizer) VisitCharLit(e *ast.CharLit) {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeString, nil))
}
func (t *semanticTokenizer) VisitStringLit(e *ast.StringLit) {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeString, nil))
}
func (t *semanticTokenizer) VisitListLit(e *ast.ListLit) {
	if e.Values != nil {
		for _, expr := range e.Values {
			expr.Accept(t)
		}
	} else if e.Count != nil && e.Value != nil {
		e.Count.Accept(t)
		e.Value.Accept(t)
	}
}
func (t *semanticTokenizer) VisitUnaryExpr(e *ast.UnaryExpr) {
	e.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitBinaryExpr(e *ast.BinaryExpr) {
	e.Lhs.Accept(t)
	e.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitTernaryExpr(e *ast.TernaryExpr) {
	e.Lhs.Accept(t)
	e.Mid.Accept(t)
	e.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitCastExpr(e *ast.CastExpr) {
	e.Lhs.Accept(t)
}
func (t *semanticTokenizer) VisitGrouping(e *ast.Grouping) {
	e.Expr.Accept(t)
}
func (t *semanticTokenizer) VisitFuncCall(e *ast.FuncCall) {
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
			if helper.GetRangeLength(cutRange[0]) != 0 {
				t.add(newHightlightedToken(cutRange[0], protocol.SemanticTokenTypeFunction, nil))
			}
			arg.Accept(t)
			rang = token.Range{Start: cutRange[1].Start, End: rang.End}

			if i == len(e.Args)-1 && helper.GetRangeLength(cutRange[1]) != 0 {
				t.add(newHightlightedToken(cutRange[1], protocol.SemanticTokenTypeFunction, nil))
			}
		}
	} else {
		t.add(newHightlightedToken(rang, protocol.SemanticTokenTypeFunction, nil))
	}
}

func (t *semanticTokenizer) VisitBadStmt(s *ast.BadStmt) {
}
func (t *semanticTokenizer) VisitDeclStmt(s *ast.DeclStmt) {
	s.Decl.Accept(t)
}
func (t *semanticTokenizer) VisitExprStmt(s *ast.ExprStmt) {
	s.Expr.Accept(t)
}
func (t *semanticTokenizer) VisitAssignStmt(s *ast.AssignStmt) {
	if s.Token().Type == token.SPEICHERE {
		s.Rhs.Accept(t)
		s.Var.Accept(t)
		return
	}
	s.Var.Accept(t)
	s.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitBlockStmt(s *ast.BlockStmt) {
	for _, stmt := range s.Statements {
		stmt.Accept(t)
	}
}
func (t *semanticTokenizer) VisitIfStmt(s *ast.IfStmt) {
	s.Condition.Accept(t)
	if s.Then != nil {
		s.Then.Accept(t)
	}
	if s.Else != nil {
		s.Else.Accept(t)
	}
}
func (t *semanticTokenizer) VisitWhileStmt(s *ast.WhileStmt) {
	switch s.While.Type {
	case token.SOLANGE:
		s.Condition.Accept(t)
		s.Body.Accept(t)
	case token.MACHE, token.COUNT_MAL:
		s.Body.Accept(t)
		s.Condition.Accept(t)
	}
}
func (t *semanticTokenizer) VisitForStmt(s *ast.ForStmt) {
	s.Initializer.Accept(t)
	s.To.Accept(t)
	if s.StepSize != nil {
		s.StepSize.Accept(t)
	}
	s.Body.Accept(t)
}
func (t *semanticTokenizer) VisitForRangeStmt(s *ast.ForRangeStmt) {
	s.Initializer.Accept(t)
	s.In.Accept(t)
	s.Body.Accept(t)
}
func (t *semanticTokenizer) VisitReturnStmt(s *ast.ReturnStmt) {
	if s.Value == nil {
		return
	}
	s.Value.Accept(t)
}

func newHightlightedToken(rang token.Range, tokType protocol.SemanticTokenType, modifiers []protocol.SemanticTokenModifier) highlightedToken {
	if modifiers == nil {
		modifiers = make([]protocol.SemanticTokenModifier, 0)
	}
	return highlightedToken{
		line:      int(rang.Start.Line),
		column:    int(rang.Start.Column),
		length:    helper.GetRangeLength(rang),
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
