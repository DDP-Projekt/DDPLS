package main

import (
	"strings"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}

	items := make([]protocol.CompletionItem, 0)
	for _, s := range ddpTypes {
		items = append(items, protocol.CompletionItem{
			Kind:  ptr(protocol.CompletionItemKindClass),
			Label: s,
		})
	}

	for _, s := range ddpKeywords {
		items = append(items, protocol.CompletionItem{
			Kind:  ptr(protocol.CompletionItemKindKeyword),
			Label: s,
		})
	}

	visitor := &tableVisitor{
		Table: currentAst.Symbols,
		pos:   params.Position,
	}

	aliases := make([]ast.FuncAlias, 0)
	for _, stmt := range currentAst.Statements {
		if decl, ok := stmt.(*ast.DeclStmt); ok {
			if funDecl, ok := decl.Decl.(*ast.FuncDecl); ok {
				aliases = append(aliases, funDecl.Aliases...)
			}
		}
		if stmt.Token().File == currentAst.File && helper.IsInRange(stmt.GetRange(), visitor.pos) {
			stmt.Accept(visitor)
			break
		}
	}

	table := visitor.Table
	varItems := make(map[string]protocol.CompletionItem)
	for table != nil {
		for name := range table.Variables {
			if _, ok := varItems[name]; !ok {
				varItems[name] = protocol.CompletionItem{
					Kind:  ptr(protocol.CompletionItemKindVariable),
					Label: name,
				}
			}
		}

		table = table.Enclosing
	}

	for _, v := range varItems {
		items = append(items, v)
	}

	for _, alias := range aliases {
		items = append(items, aliasToCompletionItem(alias))
	}

	return items, nil
}

func aliasToCompletionItem(alias ast.FuncAlias) protocol.CompletionItem {
	insertText := strings.TrimPrefix(strings.TrimSuffix(alias.Original.Literal, "\""), "\"") // remove the ""
	return protocol.CompletionItem{
		Kind:       ptr(protocol.CompletionItemKindFunction),
		Label:      alias.Func,
		InsertText: &insertText,
		Detail:     &insertText,
		FilterText: &insertText,
	}
}

func ptr[T any](v T) *T {
	return &v
}

var ddpTypes = []string{
	"Zahl",
	"Kommazahl",
	"Boolean",
	"Text",
	"Buchstabe",
	"Zahlen Liste",
	"Kommazahlen Liste",
	"Boolean Liste",
	"Text Liste",
	"Buchstaben Liste",
}

var ddpKeywords []string

// initialize the ddp-keywords
func init() {
	ddpKeywords = make([]string, 0, len(token.KeywordMap))
	for k := range token.KeywordMap {
		if !helper.Contains(ddpTypes, k) {
			ddpKeywords = append(ddpKeywords, k)
		}
	}
}

type tableVisitor struct {
	Table *ast.SymbolTable
	pos   protocol.Position
}

func (t *tableVisitor) VisitBadDecl(d *ast.BadDecl) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitVarDecl(d *ast.VarDecl) ast.Visitor {
	if helper.IsInRange(d.InitVal.GetRange(), t.pos) {
		d.InitVal.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if d.Body != nil && helper.IsInRange(d.Body.GetRange(), t.pos) {
		d.Body.Accept(t)
	}
	return t
}

func (t *tableVisitor) VisitBadExpr(e *ast.BadExpr) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitIdent(e *ast.Ident) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitIndexing(e *ast.Indexing) ast.Visitor {
	if helper.IsInRange(e.Index.GetRange(), t.pos) {
		return e.Index.Accept(t)
	}
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		return e.Lhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitFloatLit(e *ast.FloatLit) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitBoolLit(e *ast.BoolLit) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitCharLit(e *ast.CharLit) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitStringLit(e *ast.StringLit) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitListLit(e *ast.ListLit) ast.Visitor {
	if e.Values != nil {
		for _, expr := range e.Values {
			if helper.IsInRange(expr.GetRange(), t.pos) {
				return expr.Accept(t)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if helper.IsInRange(e.Count.GetRange(), t.pos) {
			return e.Count.Accept(t)
		}
		if helper.IsInRange(e.Value.GetRange(), t.pos) {
			return e.Value.Accept(t)
		}
	}
	return t
}
func (t *tableVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	if helper.IsInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	if helper.IsInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	if helper.IsInRange(e.Mid.GetRange(), t.pos) {
		e.Mid.Accept(t)
	}
	if helper.IsInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	if helper.IsInRange(e.Expr.GetRange(), t.pos) {
		e.Expr.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), t.pos) {
				return expr.Accept(t)
			}
		}
	}
	return t
}

func (t *tableVisitor) VisitBadStmt(s *ast.BadStmt) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitDeclStmt(s *ast.DeclStmt) ast.Visitor {
	return s.Decl.Accept(t)
}
func (t *tableVisitor) VisitExprStmt(s *ast.ExprStmt) ast.Visitor {
	return s.Expr.Accept(t)
}
func (t *tableVisitor) VisitAssignStmt(s *ast.AssignStmt) ast.Visitor {
	if helper.IsInRange(s.Var.GetRange(), t.pos) {
		return s.Var.Accept(t)
	}
	if helper.IsInRange(s.Rhs.GetRange(), t.pos) {
		return s.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	t.Table = s.Symbols
	for _, stmt := range s.Statements {
		if helper.IsInRange(stmt.GetRange(), t.pos) {
			return stmt.Accept(t)
		}
	}
	return t
}
func (t *tableVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	if helper.IsInRange(s.Condition.GetRange(), t.pos) {
		return s.Condition.Accept(t)
	}
	if helper.IsInRange(s.Then.GetRange(), t.pos) {
		return s.Then.Accept(t)
	}
	if s.Else != nil && helper.IsInRange(s.Else.GetRange(), t.pos) {
		return s.Else.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	if helper.IsInRange(s.Condition.GetRange(), t.pos) {
		return s.Condition.Accept(t)
	}
	if helper.IsInRange(s.Body.GetRange(), t.pos) {
		return s.Body.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), t.pos) {
		return s.Initializer.Accept(t)
	}
	if helper.IsInRange(s.To.GetRange(), t.pos) {
		return s.To.Accept(t)
	}
	if s.StepSize != nil && helper.IsInRange(s.StepSize.GetRange(), t.pos) {
		return s.StepSize.Accept(t)
	}
	if helper.IsInRange(s.Body.GetRange(), t.pos) {
		return s.Body.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), t.pos) {
		return s.Initializer.Accept(t)
	}
	if helper.IsInRange(s.In.GetRange(), t.pos) {
		return s.In.Accept(t)
	}
	if helper.IsInRange(s.Body.GetRange(), t.pos) {
		return s.Body.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(t)
}
func (t *tableVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value != nil && helper.IsInRange(s.Value.GetRange(), t.pos) {
		return s.Value.Accept(t)
	}
	return t
}
