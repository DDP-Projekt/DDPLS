package main

import (
	"errors"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

var currentAst *ast.Ast
var currentTokens []token.Token

func parse() (err error) {
	activeDoc, ok := getDocument(activeDocument)
	if !ok {
		return errors.New("activeDocument not in document map")
	}
	currentTokens, err = scanner.ScanSource(activeDocument, []byte(activeDoc.Content), func(string) {}, scanner.ModeStrictCapitalization)
	if err != nil {
		return err
	}
	currentAst = parser.ParseTokens(currentTokens, func(string) {})
	return nil
}
