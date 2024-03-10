package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var kind = protocol.DocumentHighlightKindText

func CreateTextDocumentDocumentHighlight(dm *documents.DocumentManager) protocol.TextDocumentDocumentHighlightFunc {
	return func(context *glsp.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
		act, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("%s not in document map", params.TextDocument.URI)
		}

		highlighter := &highlighter{
			pos:        params.Position,
			searchMode: true,
		}

		ast.VisitAst(act.Module.Ast, highlighter)

		highlighter.searchMode = false
		ast.VisitAst(act.Module.Ast, highlighter)

		return highlighter.highlightList, nil
	}
}

type highlighter struct {
	pos           protocol.Position
	searchMode    bool
	decl          ast.Declaration
	highlightList []protocol.DocumentHighlight
}

func (r *highlighter) BaseVisitor() {}

func (r *highlighter) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	if r.searchMode && helper.IsInRange(d.NameTok.Range, r.pos) {
		r.decl = d
		return ast.VisitBreak
	}

	if r.decl == d {
		r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
			Range: helper.ToProtocolRange(d.NameTok.Range),
			Kind:  &kind,
		})
	}

	return ast.VisitRecurse
}

func (r *highlighter) VisitIdent(d *ast.Ident) ast.VisitResult {
	if r.searchMode && helper.IsInRange(d.GetRange(), r.pos) {
		r.decl = d.Declaration
		return ast.VisitBreak
	}

	if r.decl == d.Declaration {
		r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
			Range: helper.ToProtocolRange(d.GetRange()),
			Kind:  &kind,
		})
	}

	return ast.VisitRecurse
}

func (r *highlighter) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	if r.searchMode {
		if helper.IsInRange(d.NameTok.Range, r.pos) {
			r.decl = d
			return ast.VisitBreak
		}

		for _, name := range d.ParamNames {
			if helper.IsInRange(name.Range, r.pos) {
				if d.Body == nil {
					return ast.VisitRecurse
				}
				decl, _, _ := d.Body.Symbols.LookupDecl(name.Literal)
				r.decl = decl
				return ast.VisitBreak
			}
		}

		for _, alias := range d.Aliases {
			for _, aliasTokens := range alias.Tokens {
				if helper.IsInRange(aliasTokens.Range, r.pos) {
					decl, _, _ := d.Body.Symbols.LookupDecl(aliasTokens.Literal[1 : len(aliasTokens.Literal)-1])
					r.decl = decl
					return ast.VisitBreak
				}
			}
		}
	}

	if r.decl == d {
		r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
			Range: helper.ToProtocolRange(d.NameTok.Range),
			Kind:  &kind,
		})
	}

	for _, name := range d.ParamNames {
		if d.Body == nil {
			return ast.VisitRecurse
		}

		decl, _, _ := d.Body.Symbols.LookupDecl(name.Literal)
		if decl != r.decl {
			continue
		}

		r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
			Range: helper.ToProtocolRange(decl.GetRange()),
			Kind:  &kind,
		})

		for _, alias := range d.Aliases {
			for _, aliasToken := range alias.Tokens {
				if !(aliasToken.Type == token.ALIAS_PARAMETER && aliasToken.Literal == "<"+decl.Name()+">") {
					continue
				}

				r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
					Range: helper.ToProtocolRange(token.Range{
						Start: token.Position{
							Line:   aliasToken.Range.Start.Line,
							Column: aliasToken.Range.Start.Column + 2,
						},
						End: aliasToken.Range.End,
					}),
					Kind: &kind,
				})
			}
		}
	}

	return ast.VisitRecurse
}

func (r *highlighter) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	if r.searchMode {
		for _, field := range d.Fields {
			if helper.IsInRange(field.GetRange(), r.pos) {
				r.decl = field
				return ast.VisitBreak
			}

			for _, alias := range d.Aliases {
				for _, tokens := range alias.Tokens {
					if !(tokens.Type == token.ALIAS_PARAMETER && tokens.Literal == "<"+field.Name()+">") {
						continue
					}

					if helper.IsInRange(tokens.Range, r.pos) {
						r.decl = field
						return ast.VisitBreak
					}
				}
			}
		}
	}

	for _, field := range d.Fields {
		if r.decl != field {
			continue
		}

		for _, alias := range d.Aliases {
			for _, aliasToken := range alias.Tokens {
				if !(aliasToken.Type == token.ALIAS_PARAMETER && aliasToken.Literal == "<"+field.Name()+">") {
					continue
				}

				r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
					Range: helper.ToProtocolRange(token.Range{
						Start: token.Position{
							Line:   aliasToken.Range.Start.Line,
							Column: aliasToken.Range.Start.Column + 2,
						},
						End: aliasToken.Range.End,
					}),
				})
			}
		}
	}

	return ast.VisitRecurse
}
