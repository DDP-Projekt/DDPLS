package parse

import (
	"fmt"
	"sync"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"

	"github.com/DDP-Projekt/DDPLS/documents"
)

var (
	// the resulting Ast of the activeDocument
	// should be copied on start of use to make sure
	// it doesn't change while being used
	currentAst *ast.Ast
	parseMutex = sync.Mutex{}
)

func Ast() *ast.Ast {
	return currentAst
}

func WithErrorHandler(errHndl ddperror.Handler) (*ast.Ast, error) {
	return parse(errHndl)
}

func WithoutHandler() (*ast.Ast, error) {
	return parse(ddperror.EmptyHandler)
}

// concurrency-safe re-parsing of currentAst
func parse(errHndl ddperror.Handler) (_ *ast.Ast, err error) {
	parseMutex.Lock()
	defer parseMutex.Unlock()

	activeDoc, ok := documents.Get(documents.Active)
	if !ok {
		return nil, fmt.Errorf("%s not in document map", documents.Active)
	}

	currentAst, err = parser.ParseSource(activeDoc.Path, []byte(activeDoc.Content), errHndl)
	if err != nil {
		return nil, err
	}
	return currentAst, nil
}
