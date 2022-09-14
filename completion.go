package main

import (
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
	activeDocument = params.TextDocument.URI
	if err := parse(func(token.Token, string) {}); err != nil {
		log.Errorf("parser error: %s", err)
		return nil, err
	}
	currentAst := currentAst

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

	for _, stmt := range currentAst.Statements {
		if stmt.Token().File == currentAst.File && isInRange(stmt.GetRange(), visitor.pos) {
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

	return items, nil
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
		if !contains(ddpTypes, k) {
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
	if isInRange(d.InitVal.GetRange(), t.pos) {
		d.InitVal.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitFuncDecl(d *ast.FuncDecl) ast.Visitor {
	if d.Body != nil && isInRange(d.Body.GetRange(), t.pos) {
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
	if isInRange(e.Index.GetRange(), t.pos) {
		return e.Index.Accept(t)
	}
	if isInRange(e.Lhs.GetRange(), t.pos) {
		return e.Lhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitIntLit(e *ast.IntLit) ast.Visitor {
	return t
}
func (t *tableVisitor) VisitFLoatLit(e *ast.FloatLit) ast.Visitor {
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
			if isInRange(expr.GetRange(), t.pos) {
				return expr.Accept(t)
			}
		}
	} else if e.Count != nil && e.Value != nil {
		if isInRange(e.Count.GetRange(), t.pos) {
			return e.Count.Accept(t)
		}
		if isInRange(e.Value.GetRange(), t.pos) {
			return e.Value.Accept(t)
		}
	}
	return t
}
func (t *tableVisitor) VisitUnaryExpr(e *ast.UnaryExpr) ast.Visitor {
	if isInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitBinaryExpr(e *ast.BinaryExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	if isInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitTernaryExpr(e *ast.TernaryExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	if isInRange(e.Mid.GetRange(), t.pos) {
		e.Mid.Accept(t)
	}
	if isInRange(e.Rhs.GetRange(), t.pos) {
		e.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitCastExpr(e *ast.CastExpr) ast.Visitor {
	if isInRange(e.Lhs.GetRange(), t.pos) {
		e.Lhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitGrouping(e *ast.Grouping) ast.Visitor {
	if isInRange(e.Expr.GetRange(), t.pos) {
		e.Expr.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitFuncCall(e *ast.FuncCall) ast.Visitor {
	if len(e.Args) != 0 {
		for _, expr := range e.Args {
			if isInRange(expr.GetRange(), t.pos) {
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
	if isInRange(s.Var.GetRange(), t.pos) {
		return s.Var.Accept(t)
	}
	if isInRange(s.Rhs.GetRange(), t.pos) {
		return s.Rhs.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.Visitor {
	t.Table = s.Symbols
	for _, stmt := range s.Statements {
		if isInRange(stmt.GetRange(), t.pos) {
			return stmt.Accept(t)
		}
	}
	return t
}
func (t *tableVisitor) VisitIfStmt(s *ast.IfStmt) ast.Visitor {
	if isInRange(s.Condition.GetRange(), t.pos) {
		return s.Condition.Accept(t)
	}
	if isInRange(s.Then.GetRange(), t.pos) {
		return s.Then.Accept(t)
	}
	if s.Else != nil && isInRange(s.Else.GetRange(), t.pos) {
		return s.Else.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitWhileStmt(s *ast.WhileStmt) ast.Visitor {
	if isInRange(s.Condition.GetRange(), t.pos) {
		return s.Condition.Accept(t)
	}
	if isInRange(s.Body.GetRange(), t.pos) {
		return s.Body.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitForStmt(s *ast.ForStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if isInRange(s.Initializer.GetRange(), t.pos) {
		return s.Initializer.Accept(t)
	}
	if isInRange(s.To.GetRange(), t.pos) {
		return s.To.Accept(t)
	}
	if s.StepSize != nil && isInRange(s.StepSize.GetRange(), t.pos) {
		return s.StepSize.Accept(t)
	}
	if isInRange(s.Body.GetRange(), t.pos) {
		return s.Body.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitForRangeStmt(s *ast.ForRangeStmt) ast.Visitor {
	// TODO: fix h.currentSymbols
	if isInRange(s.Initializer.GetRange(), t.pos) {
		return s.Initializer.Accept(t)
	}
	if isInRange(s.In.GetRange(), t.pos) {
		return s.In.Accept(t)
	}
	if isInRange(s.Body.GetRange(), t.pos) {
		return s.Body.Accept(t)
	}
	return t
}
func (t *tableVisitor) VisitFuncCallStmt(s *ast.FuncCallStmt) ast.Visitor {
	return s.Call.Accept(t)
}
func (t *tableVisitor) VisitReturnStmt(s *ast.ReturnStmt) ast.Visitor {
	if s.Value != nil && isInRange(s.Value.GetRange(), t.pos) {
		return s.Value.Accept(t)
	}
	return t
}
