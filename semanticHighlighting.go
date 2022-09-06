package main

import (
	"sort"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentSemanticTokensFull(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	activeDocument = params.TextDocument.URI
	if err := parse(func(token.Token, string) {}); err != nil {
		log.Errorf("parser error: %s", err)
		return nil, err
	}

	currentAst := currentAst
	tokenizer := &semanticTokenizer{}

	path, err := uriToPath(activeDocument)
	if err != nil {
		log.Warningf("url.ParseRequestURI: %s", err)
	}

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

func (t *semanticTokenizer) add(tok highlightedToken) {
	t.tokens = append(t.tokens, tok)
}

func (t *semanticTokenizer) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	return t
}
func (t *semanticTokenizer) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	t.add(newHightlightedToken(token.NewRange(d.Name, d.Name), protocol.SemanticTokenTypeVariable, nil))
	return d.InitVal.Accept(t)
}
func (t *semanticTokenizer) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	t.add(newHightlightedToken(token.NewRange(d.Name, d.Name), protocol.SemanticTokenTypeVariable, nil))
	for _, param := range d.ParamNames {
		t.add(newHightlightedToken(token.NewRange(param, param), protocol.SemanticTokenTypeParameter, nil))
	}
	if d.Body != nil {
		d.Body.Accept(t)
	}
	return t
}

func (t *semanticTokenizer) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	return t
}
func (t *semanticTokenizer) VisitIdent(e *ast.Ident) ast.Visitor {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeVariable, nil))
	return t
}
func (t *semanticTokenizer) VisitIndexing(e *ast.Indexing) ast.Visitor {
	e.Lhs.Accept(t)
	return e.Index.Accept(t)
}
func (t *semanticTokenizer) VisitIntLit(e *ast.IntLit) ast.Visitor {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeNumber, nil))
	return t
}
func (t *semanticTokenizer) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeNumber, nil))
	return t
}
func (t *semanticTokenizer) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return t
}
func (t *semanticTokenizer) VisitCharLit(e *ast.CharLit) ast.Visitor {
	t.add(newHightlightedToken(e.GetRange(), protocol.SemanticTokenTypeString, nil))
	return t
}
func (t *semanticTokenizer) VisitStringLit(e *ast.StringLit) ast.Visitor {
	rang := e.GetRange()
	rang.End.Column -= 1
	t.add(newHightlightedToken(rang, protocol.SemanticTokenTypeString, nil))
	return t
}
func (t *semanticTokenizer) VisitListLit(e *ast.ListLit) ast.Visitor {
	if e.Values != nil {
		for _, expr := range e.Values {
			expr.Accept(t)
		}
	} else if e.Count != nil && e.Value != nil {
		e.Count.Accept(t)
		e.Value.Accept(t)
	}
	return t
}
func (t *semanticTokenizer) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	return e.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	e.Lhs.Accept(t)
	return e.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	e.Lhs.Accept(t)
	e.Mid.Accept(t)
	return e.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	return e.Lhs.Accept(t)
}
func (t *semanticTokenizer) VisitGrouping(e *ast.Grouping) ast.Visitor {
	return e.Expr.Accept(t)
}
func (t *semanticTokenizer) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
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
			cutRange := cutRangeOut(rang, argRange)
			if getRangeLength(cutRange[0]) != 0 {
				t.add(newHightlightedToken(cutRange[0], protocol.SemanticTokenTypeFunction, nil))
			}
			arg.Accept(t)
			rang = token.Range{Start: cutRange[1].Start, End: rang.End}

			if i == len(e.Args)-1 && getRangeLength(cutRange[1]) != 0 {
				t.add(newHightlightedToken(cutRange[1], protocol.SemanticTokenTypeFunction, nil))
			}
		}
	} else {
		t.add(newHightlightedToken(rang, protocol.SemanticTokenTypeFunction, nil))
	}
	return t
}

func (t *semanticTokenizer) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	return t
}
func (t *semanticTokenizer) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(t)
}
func (t *semanticTokenizer) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(t)
}
func (t *semanticTokenizer) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	if s.Token().Type == token.SPEICHERE {
		s.Rhs.Accept(t)
		return s.Var.Accept(t)
	}
	s.Var.Accept(t)
	return s.Rhs.Accept(t)
}
func (t *semanticTokenizer) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	for _, stmt := range s.Statements {
		stmt.Accept(t)
	}
	return t
}
func (t *semanticTokenizer) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	s.Condition.Accept(t)
	if s.Then != nil {
		s.Then.Accept(t)
	}
	if s.Else != nil {
		s.Else.Accept(t)
	}
	return t
}
func (t *semanticTokenizer) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	switch s.While.Type {
	case token.SOLANGE:
		s.Condition.Accept(t)
		s.Body.Accept(t)
	case token.MACHE, token.COUNT_MAL:
		s.Body.Accept(t)
		s.Condition.Accept(t)
	}
	return t
}
func (t *semanticTokenizer) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	s.Initializer.Accept(t)
	s.To.Accept(t)
	if s.StepSize != nil {
		s.StepSize.Accept(t)
	}
	return s.Body.Accept(t)
}
func (t *semanticTokenizer) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	s.Initializer.Accept(t)
	s.In.Accept(t)
	return s.Body.Accept(t)
}
func (t *semanticTokenizer) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(t)
}
func (t *semanticTokenizer) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value == nil {
		return t
	}
	return s.Value.Accept(t)
}

func newHightlightedToken(rang token.Range, tokType protocol.SemanticTokenType, modifiers []protocol.SemanticTokenModifier) highlightedToken {
	if modifiers == nil {
		modifiers = make([]protocol.SemanticTokenModifier, 0)
	}
	return highlightedToken{
		line:      rang.Start.Line,
		column:    rang.Start.Column - 1,
		length:    getRangeLength(rang),
		tokenType: tokType,
		modifiers: modifiers,
	}
}

func getRangeLength(rang token.Range) int {
	if rang.Start.Line == rang.End.Line {
		return rang.End.Column - rang.Start.Column
	}
	doc, _ := getDocument(activeDocument)
	lines := strings.Split(doc.Content, "\n")
	length := len(lines[rang.Start.Line-1][rang.Start.Column-1:])
	for i := rang.Start.Line; i < rang.End.Line-1; i++ {
		length += len(lines[i])
	}
	length += len(lines[rang.End.Line-1][:rang.End.Column-2])
	return length
}

// returns two new ranges, constructed by cutting innerRange out of wholeRange
// innerRange must be completely contained in wholeRange
func cutRangeOut(wholeRange, innerRange token.Range) []token.Range {
	return []token.Range{
		{
			Start: wholeRange.Start,
			End:   innerRange.Start,
		},
		{
			Start: innerRange.End,
			End:   wholeRange.End,
		},
	}
}

var allTokenTypes = []protocol.SemanticTokenType{
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
	for i, t := range allTokenTypes {
		if t == tokenType {
			return protocol.UInteger(i)
		}
	}
	return 0
}

var allTokenModifiers = []protocol.SemanticTokenModifier{
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
