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

func CreateTextDocumentDocumentHighlight(dm *documents.DocumentManager) protocol.TextDocumentDocumentHighlightFunc {
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
		act, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("%s not in document map", params.TextDocument.URI)
		}

		highlighter := &highlighter{
			pos:        params.Position,
			searchMode: true,
		}

		ast.VisitModule(act.Module, highlighter)

		highlighter.searchMode = false
		ast.VisitModule(act.Module, highlighter)

		return highlighter.highlightList, nil
	})
}

type highlighter struct {
	pos           protocol.Position
	searchMode    bool
	decl          ast.Declaration
	highlightList []protocol.DocumentHighlight
}

var (
	_ ast.Visitor            = (*highlighter)(nil)
	_ ast.ConditionalVisitor = (*highlighter)(nil)
)

func (r *highlighter) Visitor() {}

func (r *highlighter) ShouldVisit(node ast.Node) bool {
	if !r.searchMode {
		return true
	}
	return helper.IsInRange(node.GetRange(), r.pos)
}

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
		if d.Body == nil {
			return ast.VisitRecurse
		}

		for i := range d.Parameters {
			name := d.Parameters[i].Name
			if helper.IsInRange(name.Range, r.pos) {
				r.decl, _, _ = d.Body.Symbols.LookupDecl(name.Literal)
				return ast.VisitBreak
			}
		}

		for _, alias := range d.Aliases {
			for _, aliasToken := range alias.Tokens {
				if aliasToken.Type != token.ALIAS_PARAMETER {
					continue
				}

				if helper.IsInRange(aliasToken.Range, r.pos) {
					r.decl, _, _ = d.Body.Symbols.LookupDecl(aliasToken.Literal[1 : len(aliasToken.Literal)-1])
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

	for i := range d.Parameters {
		if d.Body == nil {
			return ast.VisitRecurse
		}

		decl, _, _ := d.Body.Symbols.LookupDecl(d.Parameters[i].Name.Literal)
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
