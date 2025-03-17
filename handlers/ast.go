package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type AstRequest struct {
	Path string `json:"path"`
}

func CreateAstRequestHandler(dm *documents.DocumentManager) protocol.CustomRequestFunc {
	return func(context *glsp.Context, params json.RawMessage) (any, error) {
		var req AstRequest
		json.Unmarshal(params, &req)

		act, ok := dm.Get(req.Path)
		if !ok {
			return nil, fmt.Errorf("%s not in document map", req.Path)
		}

		items := make([]TreeItem, 0)
		for _, v := range act.Module.Ast.Statements {
			items = append(items, makeTreeNode(v))
		}
		return items, nil
	}
}

type TreeItem struct {
	Label            string     `json:"label"`
	Children         []TreeItem `json:"children"`
	CollapsibleState int        `json:"collapsibleState"`
	Description      string     `json:"description"`
	IconId           string     `json:"iconId"`
	Tooltip          string     `json:"tooltip"`
}

func NewDataItem(label string, data string, children []TreeItem) TreeItem {
	childrenState := 0
	if len(children) > 0 {
		childrenState = 1
	}

	return TreeItem{
		Label:            label,
		Description:      data,
		Children:         children,
		CollapsibleState: childrenState,
		IconId:           "",
		Tooltip:          "",
	}
}

func NewNodeItem(node ast.Node, description string, children []TreeItem, iconID string) TreeItem {
	childrenState := 0
	if len(children) > 0 {
		childrenState = 1
	}

	rang := node.GetRange()
	return TreeItem{
		Label:            node.String(),
		Children:         children,
		CollapsibleState: childrenState,
		Description:      description,
		IconId:           iconID,
		Tooltip:          fmt.Sprintf("Start: [%d, %d]\nEnd: [%d, %d]", rang.Start.Line, rang.Start.Column, rang.End.Line, rang.End.Column),
	}
}

func makeTreeNode(node ast.Node) TreeItem {
	if node == nil {
		panic("AAH NIL!!")
	}

	switch node := node.(type) {
	case *ast.BadExpr:
		return NewNodeItem(node, node.Err.Msg, nil, "error")
	case *ast.Ident:
		return NewNodeItem(node, node.Literal.Literal, nil, "symbol-variable")
	case *ast.Indexing:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Lhs),
			makeTreeNode(node.Index),
		}, "symbol-array")

	case *ast.FieldAccess:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Field),
			makeTreeNode(node.Rhs),
		}, "symbol-field")
	case *ast.IntLit:
		return NewNodeItem(node, fmt.Sprintf("%d", node.Value), nil, "symbol-number")
	case *ast.FloatLit:
		return NewNodeItem(node, fmt.Sprintf("%f", node.Value), nil, "symbol-number")
	case *ast.BoolLit:
		return NewNodeItem(node, fmt.Sprintf("%v", node.Value), nil, "breakpoints-activate")
	case *ast.CharLit:
		return NewNodeItem(node, fmt.Sprintf("%c", node.Value), nil, "text-size")
	case *ast.StringLit:
		return NewNodeItem(node, node.Literal.Literal, nil, "symbol-string")
	case *ast.ListLit:
		if node.Count != nil {
			return NewNodeItem(node, node.Type.String(), []TreeItem{
				makeTreeNode(node.Count),
				makeTreeNode(node.Value),
			}, "symbol-array")
		}

		vals := make([]TreeItem, 0)
		for _, v := range node.Values {
			vals = append(vals, makeTreeNode(v))
		}

		return NewNodeItem(node, node.Type.String(), vals, "symbol-array")
	case *ast.UnaryExpr:
		children := []TreeItem{
			makeTreeNode(node.Rhs),
		}
		if node.OverloadedBy != nil {
			children = append(children, NewDataItem("OverloadedBy", node.OverloadedBy.Decl.Name(), nil))
		}

		return NewNodeItem(node, node.Operator.String(), children, "symbol-operator")
	case *ast.BinaryExpr:
		children := []TreeItem{
			makeTreeNode(node.Lhs),
			makeTreeNode(node.Rhs),
		}
		if node.OverloadedBy != nil {
			children = append(children, NewDataItem("OverloadedBy", node.OverloadedBy.Decl.Name(), nil))
		}

		return NewNodeItem(node, node.Operator.String(), children, "symbol-operator")
	case *ast.TernaryExpr:
		children := []TreeItem{
			makeTreeNode(node.Lhs),
			makeTreeNode(node.Mid),
			makeTreeNode(node.Rhs),
		}
		if node.OverloadedBy != nil {
			children = append(children, NewDataItem("OverloadedBy", node.OverloadedBy.Decl.Name(), nil))
		}

		return NewNodeItem(node, node.Operator.String(), children, "symbol-operator")
	case *ast.CastExpr:
		children := []TreeItem{
			makeTreeNode(node.Lhs),
		}
		if node.OverloadedBy != nil {
			children = append(children, NewDataItem("OverloadedBy", node.OverloadedBy.Decl.Name(), nil))
		}

		return NewNodeItem(node, node.TargetType.String(), children, "symbol-operator")
	case *ast.TypeOpExpr:
		return NewNodeItem(node, node.Operator.String(), nil, "symbol-operator")
	case *ast.TypeCheck:
		return NewNodeItem(node, node.CheckType.String(), []TreeItem{
			makeTreeNode(node.Lhs),
		}, "symbol-operator")

	case *ast.Grouping:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Expr),
		}, "symbol-namespace")
	case *ast.FuncCall:
		args := make([]TreeItem, 0)
		for _, v := range node.Args {
			args = append(args, makeTreeNode(v))
		}

		return NewNodeItem(node, node.Name, args, "symbol-function")
	case *ast.StructLiteral:
		args := make([]TreeItem, 0)
		for _, v := range node.Args {
			args = append(args, makeTreeNode(v))
		}

		return NewNodeItem(node, node.Struct.Name(), args, "symbol-constructor")

	case *ast.BadDecl:
		return NewNodeItem(node, node.Err.Msg, nil, "error")
	case *ast.ConstDecl:
		return NewNodeItem(node, node.Name(), []TreeItem{
			makeTreeNode(node.Val),
			NewDataItem("Type", node.Type.String(), nil),
			NewDataItem("IsPublic", fmt.Sprintf("%v", node.IsPublic), nil),
		}, "symbol-constant")
	case *ast.VarDecl:
		return NewNodeItem(node, node.Name(), []TreeItem{
			makeTreeNode(node.InitVal),
			NewDataItem("Type", node.Type.String(), nil),
			NewDataItem("IsPublic", fmt.Sprintf("%v", node.IsPublic), nil),
			NewDataItem("IsExternVisible", fmt.Sprintf("%v", node.IsExternVisible), nil),
		}, "symbol-variable")
	case *ast.FuncDecl:
		children := make([]TreeItem, 0)
		aliase := make([]TreeItem, 0)
		for _, v := range node.Aliases {
			params := make([]TreeItem, 0)
			for paramName, paramType := range v.Args {
				if paramType.IsReference && ddptypes.IsAny(paramType.Type) {
					continue // TODO remove after fix 9d13193bdc4e3a563f24565540243a2631d2824a is merged
				}
				params = append(params, NewDataItem(paramName, paramType.String(), nil))
			}

			aliase = append(aliase, NewDataItem("Alias", node.Name(), []TreeItem{
				NewDataItem("Negated", fmt.Sprintf("%v", v.Negated), nil),
				NewDataItem("Params", "", params),
			}))

		}
		children = append(children, NewDataItem("Aliase", "", aliase))
		children = append(children, NewDataItem("IsPublic", fmt.Sprintf("%v", node.IsPublic), nil))
		children = append(children, NewDataItem("IsExternVisible", fmt.Sprintf("%v", node.IsExternVisible), nil))

		if node.Body == nil {
			children = append(children, NewDataItem("ExternFile", node.ExternFile.Literal, nil))

			return NewNodeItem(node, node.Name(), children, "symbol-function")
		}

		if node.Def != nil {
			children = append(children, makeTreeNode(node.Def))

			return NewNodeItem(node, node.Name(), children, "symbol-function")
		}

		children = append(children, makeTreeNode(node.Body))

		return NewNodeItem(node, node.Name(), children, "symbol-function")
	case *ast.FuncDef:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Body),
		}, "symbol-function")
	case *ast.StructDecl:
		children := make([]TreeItem, 0)

		for _, v := range node.Fields {
			children = append(children, makeTreeNode(v))
		}

		aliase := make([]TreeItem, 0)
		for _, v := range node.Aliases {
			params := make([]TreeItem, 0)
			for paramName, paramType := range v.Args {
				params = append(params, NewDataItem(paramName, paramType.String(), nil))
			}

			aliase = append(aliase, NewDataItem("Alias", node.Name(), []TreeItem{
				NewDataItem("Params", "", params),
			}))

		}
		children = append(children, NewDataItem("Aliase", "", aliase))
		children = append(children, NewDataItem("IsPublic", fmt.Sprintf("%v", node.IsPublic), nil))

		return NewNodeItem(node, node.Name(), children, "symbol-struct")
	case *ast.TypeAliasDecl:
		return NewNodeItem(node, node.Name(), nil, "replace")
	case *ast.TypeDefDecl:
		return NewNodeItem(node, node.Name(), nil, "replace")
	case *ast.BadStmt:
		return NewNodeItem(node, node.Err.Msg, nil, "error")
	case *ast.DeclStmt:
		if node.Decl == nil {
			return NewNodeItem(node, "", nil, "symbol-class")
		}

		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Decl),
		}, "symbol-class")
	case *ast.ExprStmt:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Expr),
		}, "symbol-misc")
	case *ast.ImportStmt:
		imports := make([]TreeItem, 0)
		if !node.IsDirectoryImport {
			for _, v := range node.SingleModule().Ast.Statements {
				switch v.(type) {
				case *ast.ImportStmt:
					continue // for efficiency
				}
				imports = append(imports, makeTreeNode(v))
			}
		}

		return NewNodeItem(node, node.FileName.Literal, imports, "library")
	case *ast.AssignStmt:
		return NewNodeItem(node, node.RhsType.String(), []TreeItem{
			makeTreeNode(node.Var),
			makeTreeNode(node.Rhs),
		}, "symbol-value")
	case *ast.BlockStmt:
		children := make([]TreeItem, 0)
		for _, v := range node.Statements {
			children = append(children, makeTreeNode(v))
		}

		return NewNodeItem(node, "", children, "symbol-namespace")
	case *ast.IfStmt:
		children := []TreeItem{
			makeTreeNode(node.Condition),
			makeTreeNode(node.Then),
		}
		if node.Else != nil {
			children = append(children, makeTreeNode(node.Else))
		}

		return NewNodeItem(node, "", children, "repo-forked")
	case *ast.WhileStmt:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Condition),
			makeTreeNode(node.Body),
		}, "sync")
	case *ast.ForStmt:
		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Initializer),
			makeTreeNode(node.To),
			makeTreeNode(node.StepSize),
			makeTreeNode(node.Body),
		}, "sync")
	case *ast.ForRangeStmt:
		children := []TreeItem{
			makeTreeNode(node.Initializer),
			makeTreeNode(node.In),
			makeTreeNode(node.Body),
		}
		if node.Index != nil {
			children = append(children, makeTreeNode(node.Index))
		}

		return NewNodeItem(node, "", children, "sync")
	case *ast.BreakContinueStmt:
		return NewNodeItem(node, node.Tok.String(), nil, "sync-ignored")
	case *ast.ReturnStmt:
		if node.Value == nil {
			return NewNodeItem(node, "", nil, "newline")
		}

		return NewNodeItem(node, "", []TreeItem{
			makeTreeNode(node.Value),
		}, "newline")
	case *ast.TodoStmt:
		return NewNodeItem(node, "", nil, "ellipsis")
	}

	panic("you missed one")
}
