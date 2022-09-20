package handlers

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

func TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
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

func (t *tableVisitor) VisitBadDecl(d *ast.BadDecl) {
}
func (t *tableVisitor) VisitVarDecl(d *ast.VarDecl) {
	if helper.IsInRange(d.InitVal.GetRange(), t.pos) {
		d.InitVal.Accept(t)
	}
}
func (t *tableVisitor) VisitFuncDecl(d *ast.FuncDecl) {
	if d.Body != nil && helper.IsInRange(d.Body.GetRange(), t.pos) {
		d.Body.Accept(t)
	}
}

func (t *tableVisitor) VisitBadExpr(e *ast.BadExpr) {
}
func (t *tableVisitor) VisitIdent(e *ast.Ident) {
}
func (t *tableVisitor) VisitIndexing(e *ast.Indexing) {
	if helper.IsInRange(e.Index.GetRange(), t.pos) {
		e.Index.Accept(t)
	}
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
}
func (t *tableVisitor) VisitIntLit(e *ast.IntLit) {
}
func (t *tableVisitor) VisitFloatLit(e *ast.FloatLit) {
}
func (t *tableVisitor) VisitBoolLit(e *ast.BoolLit) {
}
func (t *tableVisitor) VisitCharLit(e *ast.CharLit) {
}
func (t *tableVisitor) VisitStringLit(e *ast.StringLit) {
}
func (t *tableVisitor) VisitListLit(e *ast.ListLit) {
	if e.Values != nil {
		for _, expr := range e.Values {
			if helper.IsInRange(expr.GetRange(), t.pos) {
				expr.Accept(t)
				return
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if helper.IsInRange(e.Count.GetRange(), t.pos) {
			e.Count.Accept(t)
			return
		}
		if helper.IsInRange(e.Value.GetRange(), t.pos) {
			e.Value.Accept(t)
			return
		}
	}
}
func (t *tableVisitor) VisitUnaryExpr(e *ast.UnaryExpr) {
	if helper.IsInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
}
func (t *tableVisitor) VisitBinaryExpr(e *ast.BinaryExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	if helper.IsInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
}
func (t *tableVisitor) VisitTernaryExpr(e *ast.TernaryExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	if helper.IsInRange(e.Mid.GetRange(), t.pos) {
		e.Mid.Accept(t)
	}
	if helper.IsInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
}
func (t *tableVisitor) VisitCastExpr(e *ast.CastExpr) {
	if helper.IsInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
}
func (t *tableVisitor) VisitGrouping(e *ast.Grouping) {
	if helper.IsInRange(e.Expr.GetRange(), t.pos) {
		e.Expr.Accept(t)
	}
}
func (t *tableVisitor) VisitFuncCall(e *ast.FuncCall) {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if helper.IsInRange(expr.GetRange(), t.pos) {
				expr.Accept(t)
				return
			}
		}
	}
}

func (t *tableVisitor) VisitBadStmt(s *ast.BadStmt) {
}
func (t *tableVisitor) VisitDeclStmt(s *ast.DeclStmt) {
	s.Decl.Accept(t)
}
func (t *tableVisitor) VisitExprStmt(s *ast.ExprStmt) {
	s.Expr.Accept(t)
}
func (t *tableVisitor) VisitAssignStmt(s *ast.AssignStmt) {
	if helper.IsInRange(s.Var.GetRange(), t.pos) {
		s.Var.Accept(t)
		return
	}
	if helper.IsInRange(s.Rhs.GetRange(), t.pos) {
		s.Rhs.Accept(t)
		return
	}
}
func (t *tableVisitor) VisitBlockStmt(s *ast.BlockStmt) {
	t.Table = s.Symbols
	for _, stmt := range s.Statements {
		if helper.IsInRange(stmt.GetRange(), t.pos) {
			stmt.Accept(t)
			return
		}
	}
}
func (t *tableVisitor) VisitIfStmt(s *ast.IfStmt) {
	if helper.IsInRange(s.Condition.GetRange(), t.pos) {
		s.Condition.Accept(t)
		return
	}
	if helper.IsInRange(s.Then.GetRange(), t.pos) {
		s.Then.Accept(t)
		return
	}
	if s.Else != nil && helper.IsInRange(s.Else.GetRange(), t.pos) {
		s.Else.Accept(t)
		return
	}
}
func (t *tableVisitor) VisitWhileStmt(s *ast.WhileStmt) {
	if helper.IsInRange(s.Condition.GetRange(), t.pos) {
		s.Condition.Accept(t)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), t.pos) {
		s.Body.Accept(t)
		return
	}
}
func (t *tableVisitor) VisitForStmt(s *ast.ForStmt) {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), t.pos) {
		s.Initializer.Accept(t)
		return
	}
	if helper.IsInRange(s.To.GetRange(), t.pos) {
		s.To.Accept(t)
		return
	}
	if s.StepSize != nil && helper.IsInRange(s.StepSize.GetRange(), t.pos) {
		s.StepSize.Accept(t)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), t.pos) {
		s.Body.Accept(t)
		return
	}
}
func (t *tableVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) {
	// TODO: fix h.currentSymbols
	if helper.IsInRange(s.Initializer.GetRange(), t.pos) {
		s.Initializer.Accept(t)
		return
	}
	if helper.IsInRange(s.In.GetRange(), t.pos) {
		s.In.Accept(t)
		return
	}
	if helper.IsInRange(s.Body.GetRange(), t.pos) {
		s.Body.Accept(t)
		return
	}
}
func (t *tableVisitor) VisitReturnStmt(s *ast.ReturnStmt) {
	if s.Value != nil && helper.IsInRange(s.Value.GetRange(), t.pos) {
		s.Value.Accept(t)
	}
}
