---
name: Language Server crash
about: Der Language Server ist abgestürzt
title: ''
labels: bug
assignees: ''

---

# Language Server absturz
Beschreibung/Weitere Informationen

## Fehlermeldung und Stack trace
Kopiere die Fehlermeldung mit dem Stack trace vom DDPLS output hier hin. Zum Beispiel:
```
panic: runtime error: index out of range [5] with length 5

goroutine 14 [running]:
github.com/DDP-Projekt/DDPLS/helper.GetRangeLength({{0xa30a20?, 0xc0002c4990?}, {0x2?, 0xa30a92?}}, 0xc000596f70?)
	D:/Hendrik/source/OtherLanguages/GoRepos/DDP-Projekt/DDPLS/helper/protocol_helper.go:33 +0x114
github.com/DDP-Projekt/DDPLS/handlers.(*semanticTokenizer).VisitFuncCall(0xc00060fc40, 0xc0005b6ea0)
	D:/Hendrik/source/OtherLanguages/GoRepos/DDP-Projekt/DDPLS/handlers/semanticHighlighting.go:190 +0x52f
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).VisitFuncCall(0xc0002c4978, 0xc0005b6ea0)
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:252 +0x5f
github.com/DDP-Projekt/Kompilierer/src/ast.(*FuncCall).Accept(0x204c62c0900?, {0xb0efe0?, 0xc0002c4978?})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/ast.go:426 +0x27
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).visit(0xc0002c4978, {0x204c62c0900, 0xc0005b6ea0})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:83 +0xbf
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).visitChildren(0x80a67a?, 0xc0?, {0xc000597438?, 0x1, 0x18?})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:96 +0x7e
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).VisitGrouping(0xc0002c4978, 0xc000540f80)
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:247 +0x9c
github.com/DDP-Projekt/Kompilierer/src/ast.(*Grouping).Accept(0x204c62c82d8?, {0xb0efe0?, 0xc0002c4978?})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/ast.go:425 +0x27
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).visit(0xc0002c4978, {0x204c62c82d8, 0xc000540f80})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:83 +0xbf
github.com/DDP-Projekt/Kompilierer/src/ast.VisitNode({0xb0a5c0?, 0xc00060fc40}, {0x204c62c82d8, 0xc000540f80}, 0x0)
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:74 +0x116
github.com/DDP-Projekt/DDPLS/handlers.(*semanticTokenizer).VisitFuncCall(0xc00060fc40, 0xc0005b6f30)
	D:/Hendrik/source/OtherLanguages/GoRepos/DDP-Projekt/DDPLS/handlers/semanticHighlighting.go:193 +0x72d
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).VisitFuncCall(0xc0002c4948, 0xc0005b6f30)
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:252 +0x5f
github.com/DDP-Projekt/Kompilierer/src/ast.(*FuncCall).Accept(0x204c62c0900?, {0xb0efe0?, 0xc0002c4948?})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/ast.go:426 +0x27
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).visit(0xc0002c4948, {0x204c62c0900, 0xc0005b6f30})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:83 +0xbf
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).visitChildren(0x80a67a?, 0xc0?, {0xc0005979f8?, 0x1, 0xb0a5c0?})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:96 +0x7e
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).VisitExprStmt(0xc0002c4948, 0xc0005a9790)
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:282 +0x9b
github.com/DDP-Projekt/Kompilierer/src/ast.(*ExprStmt).Accept(0x204c62c08c8?, {0xb0efe0?, 0xc0002c4948?})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/ast.go:581 +0x24
github.com/DDP-Projekt/Kompilierer/src/ast.(*helperVisitor).visit(0xc0002c4948, {0x204c62c08c8, 0xc0005a9790})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:83 +0xbf
github.com/DDP-Projekt/Kompilierer/src/ast.VisitAst(0xc00009e4c0, {0xb0a5c0?, 0xc00060fc40})
	C:/Users/Hendrik/Go/pkg/mod/github.com/!d!d!p-!projekt/!kompilierer@v0.2.0-alpha/src/ast/helper_visitor.go:47 +0x15a
github.com/DDP-Projekt/DDPLS/ddpls.NewDDPLS.CreateTextDocumentSemanticTokensFull.func6(0xc00001b5e0?, 0xc000432340)
	D:/Hendrik/source/OtherLanguages/GoRepos/DDP-Projekt/DDPLS/handlers/semanticHighlighting.go:31 +0xc5
github.com/tliron/glsp/protocol_3_16.(*Handler).Handle(0xc0004d0d80, 0xc00060fc00)
	C:/Users/Hendrik/Go/pkg/mod/github.com/tliron/glsp@v0.2.1/protocol_3_16/handler.go:662 +0x3802
github.com/tliron/glsp/server.(*Server).handle(0xc000321200, {0xb0bfc0?, 0xcfbfa0}, 0xc0004c6ab0, 0xc000321bc0)
	C:/Users/Hendrik/Go/pkg/mod/github.com/tliron/glsp@v0.2.1/server/handle.go:46 +0x28f
github.com/sourcegraph/jsonrpc2.(*HandlerWithErrorConfigurer).Handle(0xc000528560, {0xb0bfc0, 0xcfbfa0}, 0xc0004c6ab0, 0xc000321bc0)
	C:/Users/Hendrik/Go/pkg/mod/github.com/sourcegraph/jsonrpc2@v0.2.0/handler_with_error.go:21 +0x57
github.com/sourcegraph/jsonrpc2.(*Conn).readMessages(0xc0004c6ab0, {0xb0bfc0, 0xcfbfa0})
	C:/Users/Hendrik/Go/pkg/mod/github.com/sourcegraph/jsonrpc2@v0.2.0/conn.go:205 +0x267
created by github.com/sourcegraph/jsonrpc2.NewConn in goroutine 1
	C:/Users/Hendrik/Go/pkg/mod/github.com/sourcegraph/jsonrpc2@v0.2.0/conn.go:62 +0x236
```

## Reproduzierung
Schreibe hier die Schritte die man gehen muss um den Absturz zu Reproduzieren oder einen Beispiel Code welches den Language Server abstürzt
