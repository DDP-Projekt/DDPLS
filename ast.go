package main

import (
	"errors"
	"sync"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

var (
	// the resulting Ast of the activeDocument
	// should be copied on start of use to make sure
	// it doesn't change while being used
	currentAst *ast.Ast
	parseMutex = sync.Mutex{}
)

// concurrency-safe re-parsing of currentAst
func parse(errHndl scanner.ErrorHandler) (err error) {
	parseMutex.Lock()
	defer parseMutex.Unlock()

	activeDoc, ok := getDocument(activeDocument)
	if !ok {
		return errors.New("activeDocument not in document map")
	}

	currentAst, err = parser.ParseSource(activeDoc.Path, []byte(activeDoc.Content), errHndl)
	if err != nil {
		return err
	}
	return nil
}
