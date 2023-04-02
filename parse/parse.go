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
	currentAst   *ast.Ast
	currentError error // error result of the latest invocation of parse
	parseMutex   = sync.Mutex{}
)

func Ast() (*ast.Ast, error) {
	parseMutex.Lock()
	defer parseMutex.Unlock()
	return currentAst, currentError
}

// reparses only if documents.Active != docURI
func ReparseIfNotActive(docURI string) (*ast.Ast, error) {
	if docURI != documents.Active {
		documents.Active = docURI
		return parse(ddperror.EmptyHandler)
	}
	return Ast()
}

func WithErrorHandler(errHndl ddperror.Handler) (*ast.Ast, error) {
	return parse(errHndl)
}

func WithoutHandler() (*ast.Ast, error) {
	return parse(ddperror.EmptyHandler)
}

// concurrency-safe re-parsing of currentAst
func parse(errHndl ddperror.Handler) (*ast.Ast, error) {
	parseMutex.Lock()
	defer parseMutex.Unlock()

	activeDoc, ok := documents.Get(documents.Active)
	if !ok {
		return nil, fmt.Errorf("%s not in document map", documents.Active)
	}

	currentAst, currentError = parser.Parse(parser.Options{
		FileName:     activeDoc.Path,
		Source:       []byte(activeDoc.Content),
		ErrorHandler: errHndl,
		Tokens:       nil,
	})

	return currentAst, currentError
}
