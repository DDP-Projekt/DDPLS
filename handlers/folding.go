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
	return RecoverAnyErr(func(context *glsp.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
		var docMod *ast.Module
		if doc, ok := dm.Get(params.TextDocument.URI); !ok {
			return nil, fmt.Errorf("document not found %s", params.TextDocument.URI)
		} else {
			docMod = doc.Module
		}

		visitor := &foldingVisitor{
			foldRanges: make([]protocol.FoldingRange, 0, 8),
		}

		ast.VisitModule(docMod, visitor)

		return visitor.foldRanges, nil
	})
}

type foldingVisitor struct {
	foldRanges []protocol.FoldingRange
}

var foldingVisitor_ ast.Visitor = (*foldingVisitor)(nil)

func (*foldingVisitor) Visitor() {}

func (fold *foldingVisitor) VisitBlockStmt(s *ast.BlockStmt) ast.VisitResult {
	fold.foldRanges = append(fold.foldRanges, protocol.FoldingRange{
		StartLine: helper.ToProtocolRange(s.GetRange()).Start.Line,
		EndLine:   helper.ToProtocolRange(s.GetRange()).End.Line,
	})
	return ast.VisitRecurse
}

func (fold *foldingVisitor) VisitStructDecl(d *ast.StructDecl) ast.VisitResult {
	endRange := d.GetRange()
	if len(d.Aliases) > 0 {
		endRange = d.Aliases[0].Original.Range
	}
	fold.foldRanges = append(fold.foldRanges, protocol.FoldingRange{
		StartLine: helper.ToProtocolRange(d.GetRange()).Start.Line,
		EndLine:   helper.ToProtocolRange(endRange).Start.Line - 2,
	})
	if len(d.Aliases) > 0 {
		startRange := d.Aliases[0].Original.Range
		endRange := d.Aliases[len(d.Aliases)-1].Original.Range
		fold.foldRanges = append(fold.foldRanges, protocol.FoldingRange{
			StartLine: helper.ToProtocolRange(startRange).Start.Line - 1,
			EndLine:   helper.ToProtocolRange(endRange).Start.Line,
		})
	}
	return ast.VisitRecurse
}
