package main

import (
	"errors"
	"sync"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

var currentAst *ast.Ast
var currentTokens []token.Token

var parseMutex = sync.Mutex{}

func parse(errHndl scanner.ErrorHandler) (err error) {
	parseMutex.Lock()
	activeDoc, ok := getDocument(activeDocument)
	if !ok {
		return errors.New("activeDocument not in document map")
	}
	currentTokens, err = scanner.ScanSource(activeDocument, []byte(activeDoc.Content), errHndl, scanner.ModeStrictCapitalization)
	if err != nil {
		return err
	}
	currentAst = parser.ParseTokens(currentTokens, errHndl)
	parseMutex.Unlock()
	return nil
}

func emptyErrHndl(token.Token, string) {}
