package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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
				r.decl, _, _ = d.Body.Symbols.LookupDecl(name.Literal)
				return ast.VisitBreak
			}
		}

		for _, alias := range d.Aliases {
			for _, aliasTokens := range alias.Tokens {
				if helper.IsInRange(aliasTokens.Range, r.pos) {
					r.decl, _, _ = d.Body.Symbols.LookupDecl(aliasTokens.Literal[1 : len(aliasTokens.Literal)-1])
					return ast.VisitBreak
				}
			}
		}
	}

	if r.decl == d {
		r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
			Range: helper.ToProtocolRange(d.NameTok.Range),
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
		})

		for _, alias := range d.Aliases {
			for _, aliasToken := range alias.Tokens {
				if !helper.AliasParamNameEquals(aliasToken, decl.Name()) {
					continue
				}

				r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
					Range: helper.GetAliasParamProtocolRange(aliasToken),
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
				for _, aliasToken := range alias.Tokens {
					if !helper.AliasParamNameEquals(aliasToken, field.Name()) {
						continue
					}

					if helper.IsInRange(aliasToken.Range, r.pos) {
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
				if !helper.AliasParamNameEquals(aliasToken, field.Name()) {
					continue
				}

				r.highlightList = append(r.highlightList, protocol.DocumentHighlight{
					Range: helper.GetAliasParamProtocolRange(aliasToken),
				})
			}
		}
	}

	return ast.VisitRecurse
}
