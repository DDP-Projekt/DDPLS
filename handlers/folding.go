package handlers

import (
	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/DDPLS/parse"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TextDocumentFoldingRange(context *glsp.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	documents.Active = params.TextDocument.URI
	var currentAst *ast.Ast
	var err error
	if currentAst, err = parse.WithoutHandler(); err != nil {
		return nil, err
	}

	visitor := &foldingVisitor{
		foldRanges: nil,
		currentAst: currentAst,
	}

	ast.VisitAst(currentAst, visitor)

	return visitor.foldRanges, nil
}

type foldingVisitor struct {
	foldRanges []protocol.FoldingRange
	currentAst *ast.Ast
}

func (*foldingVisitor) BaseVisitor() {}

func (fold *foldingVisitor) VisitBlockStmt(s *ast.BlockStmt) {
	if s.Token().File == fold.currentAst.File {
		foldRange := protocol.FoldingRange{
			StartLine: helper.ToProtocolRange(s.GetRange()).Start.Line,
			EndLine:   helper.ToProtocolRange(s.GetRange()).End.Line,
		}

		fold.foldRanges = append(fold.foldRanges, foldRange)
	}
}
