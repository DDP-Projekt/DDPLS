package parse

import (
	"errors"
	"sync"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"

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

func WithErrorHandler(errHndl scanner.ErrorHandler) (*ast.Ast, error) {
	return parse(errHndl)
}

func WithoutHandler() (*ast.Ast, error) {
	return parse(func(token.Token, string) {})
}

// concurrency-safe re-parsing of currentAst
func parse(errHndl scanner.ErrorHandler) (_ *ast.Ast, err error) {
	parseMutex.Lock()
	defer parseMutex.Unlock()

	activeDoc, ok := documents.Get(documents.Active)
	if !ok {
		return nil, errors.New("activeDocument not in document map")
	}

	currentAst, err = parser.ParseSource(activeDoc.Path, []byte(activeDoc.Content), errHndl)
	if err != nil {
		return nil, err
	}
	return currentAst, nil
}
