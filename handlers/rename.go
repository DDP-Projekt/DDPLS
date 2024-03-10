package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentPrepareRename(dm *documents.DocumentManager) protocol.TextDocumentPrepareRenameFunc {
	return func(context *glsp.Context, params *protocol.PrepareRenameParams) (any, error) {
		var docMod *ast.Module
		if doc, ok := dm.Get(params.TextDocument.URI); !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		} else {
			docMod = doc.Module
		}

		preparer := renamePreparer{
			pos: params.Position,
		}

		ast.VisitModuleRec(docMod, &preparer)

		return protocol.DefaultBehavior{
			DefaultBehavior: preparer.decl != nil,
		}, nil
	}
}

type renamePreparer struct {
	decl ast.Declaration
	pos  protocol.Position
}

func (r *renamePreparer) BaseVisitor() {}

func (r *renamePreparer) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	if helper.IsInRange(d.NameTok.Range, r.pos) {
		r.decl = d
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (r *renamePreparer) VisitIdent(d *ast.Ident) ast.VisitResult {
	if helper.IsInRange(d.GetRange(), r.pos) {
		r.decl = d.Declaration
		return ast.VisitBreak
	}
	return ast.VisitRecurse
}

func (r *renamePreparer) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
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
		for _, tokens := range alias.Tokens {
			if helper.IsInRange(tokens.Range, r.pos) {
				decl, _, _ := d.Body.Symbols.LookupDecl(tokens.Literal[1 : len(tokens.Literal)-1])
				r.decl = decl
				return ast.VisitBreak
			}
		}
	}

	if helper.IsInRange(d.NameTok.Range, r.pos) {
		r.decl = d
		return ast.VisitBreak
	}

	return ast.VisitRecurse
}

func (r *renamePreparer) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
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

	return ast.VisitRecurse
}

func CreateTextDocumentRename(dm *documents.DocumentManager) protocol.TextDocumentRenameFunc {
	return func(context *glsp.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
		var docMod *ast.Module
		if doc, ok := dm.Get(params.TextDocument.URI); !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		} else {
			docMod = doc.Module
		}

		renamePreparer := renamePreparer{
			pos: params.Position,
		}

		ast.VisitModuleRec(docMod, &renamePreparer)
		if renamePreparer.decl == nil {
			return nil, fmt.Errorf("no declaration found at position")
		}

		renamer := renamer{
			newName: params.NewName,
			decl:    renamePreparer.decl,
			changes: make(map[string][]protocol.TextEdit),
			uri:     params.TextDocument.URI,
		}

		ast.VisitModuleRec(docMod, &renamer)

		edit := &protocol.WorkspaceEdit{
			Changes: renamer.changes,
		}

		return edit, nil
	}
}

type renamer struct {
	changes map[protocol.DocumentUri][]protocol.TextEdit
	newName string
	decl    ast.Declaration
	uri     protocol.DocumentUri
}

func (r *renamer) BaseVisitor() {}

func (r *renamer) VisitIdent(e *ast.Ident) ast.VisitResult {
	if e.Declaration != r.decl {
		return ast.VisitRecurse
	}

	//uri := protocol.DocumentUri(uri.FromPath(e.Declaration.Mod.FileName))
	r.changes[r.uri] = append(r.changes[r.uri], protocol.TextEdit{
		Range:   helper.ToProtocolRange(e.GetRange()),
		NewText: r.newName,
	})
	return ast.VisitRecurse
}

func (r *renamer) VisitVarDecl(d *ast.VarDecl) ast.VisitResult {
	if d != r.decl {
		return ast.VisitRecurse
	}

	//uri := protocol.DocumentUri(uri.FromPath(d.Mod.FileName))
	r.changes[r.uri] = append(r.changes[r.uri], protocol.TextEdit{
		Range:   helper.ToProtocolRange(d.NameTok.Range),
		NewText: r.newName,
	})

	return ast.VisitRecurse
}

func (r *renamer) VisitFuncDecl(d *ast.FuncDecl) ast.VisitResult {
	if d == r.decl {
		uri := protocol.DocumentUri(uri.FromPath(d.Mod.FileName))
		r.changes[uri] = append(r.changes[uri], protocol.TextEdit{
			Range:   helper.ToProtocolRange(d.NameTok.Range),
			NewText: r.newName,
		})
	}

	for _, name := range d.ParamNames {
		if d.Body == nil {
			return ast.VisitRecurse
		}

		decl, _, _ := d.Body.Symbols.LookupDecl(name.Literal)
		if decl == r.decl {
			//uri := protocol.DocumentUri(uri.FromPath(d.Mod.FileName))
			r.changes[r.uri] = append(r.changes[r.uri], protocol.TextEdit{
				Range:   helper.ToProtocolRange(name.Range),
				NewText: r.newName,
			})

			for _, alias := range d.Aliases {
				for _, aliasToken := range alias.Tokens {
					if !(aliasToken.Type == token.ALIAS_PARAMETER && aliasToken.Literal == "<"+decl.Name()+">") {
						continue
					}

					r.changes[r.uri] = append(r.changes[r.uri], protocol.TextEdit{
						Range:   helper.ToProtocolRange(getAliasRange(aliasToken)),
						NewText: r.newName,
					})
				}
			}
		}
	}

	return ast.VisitRecurse
}

func (r *renamer) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	for _, field := range d.Fields {
		if r.decl != field {
			continue
		}

		for _, alias := range d.Aliases {
			for _, aliasToken := range alias.Tokens {
				if !(aliasToken.Type == token.ALIAS_PARAMETER && aliasToken.Literal == "<"+field.Name()+">") {
					continue
				}

				r.changes[r.uri] = append(r.changes[r.uri], protocol.TextEdit{
					Range:   helper.ToProtocolRange(getAliasRange(aliasToken)),
					NewText: r.newName,
				})
			}
		}
	}

	return ast.VisitRecurse
}

func getAliasRange(aliasToken token.Token) token.Range {
	return token.Range{
		Start: token.Position{
			Line:   aliasToken.Range.Start.Line,
			Column: aliasToken.Range.Start.Column + 2,
		},
		End: aliasToken.Range.End,
	}
}
