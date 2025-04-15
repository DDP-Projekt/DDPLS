package documents

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
)

type genericsClearer struct {
	mod *ast.Module
	vis ast.FullVisitor
}

var (
	_ ast.FuncCallVisitor = (*genericsClearer)(nil)
	_ ast.VisitorSetter   = (*genericsClearer)(nil)
)

func (genericsClearer) Visitor() {}

func (c *genericsClearer) SetVisitor(vis ast.FullVisitor) {
	c.vis = vis
}

func (c *genericsClearer) VisitFuncCall(call *ast.FuncCall) ast.VisitResult {
	if ast.IsGenericInstantiation(call.Func) {
		delete(call.Func.GenericDecl.Generic.Instantiations, c.mod)
		c.vis.VisitFuncDecl(call.Func)
	}
	return ast.VisitRecurse
}
