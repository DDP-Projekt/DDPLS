package handlers

import (
	"fmt"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/DDPLS/helper"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func CreateTextDocumentFoldingRange(dm *documents.DocumentManager) protocol.TextDocumentFoldingRangeFunc {
	return func(context *glsp.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
		doc, ok := dm.Get(params.TextDocument.URI)
		if !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		}

		visitor := &foldingVisitor{
			foldRanges: make([]protocol.FoldingRange, 0),
			module:     doc.Module,
		}

		ast.VisitAst(doc.Module.Ast, visitor)

		return visitor.foldRanges, nil
	}
}

type foldingVisitor struct {
	foldRanges []protocol.FoldingRange
	module     *ast.Module
}

func (*foldingVisitor) BaseVisitor() {}

func (fold *foldingVisitor) VisitBlockStmt(s *ast.BlockStmt) {
	foldRange := protocol.FoldingRange{
		StartLine: helper.ToProtocolRange(s.GetRange()).Start.Line,
		EndLine:   helper.ToProtocolRange(s.GetRange()).End.Line,
	}

	fold.foldRanges = append(fold.foldRanges, foldRange)
}
