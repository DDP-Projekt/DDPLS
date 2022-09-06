package main

import (
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// converts a token.Range to a protocol.Range
func toProtocolRange(rang token.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(rang.Start.Line - 1),
			Character: uint32(rang.Start.Column - 1),
		},
		End: protocol.Position{
			Line:      uint32(rang.End.Line - 1),
			Character: uint32(rang.End.Column - 1),
		},
	}
}

// returns the length of a token.Range
func getRangeLength(rang token.Range) int {
	if rang.Start.Line == rang.End.Line {
		return rang.End.Column - rang.Start.Column
	}
	doc, _ := getDocument(activeDocument)
	lines := strings.Split(doc.Content, "\n")
	length := len(lines[rang.Start.Line-1][rang.Start.Column-1:])
	for i := rang.Start.Line; i < rang.End.Line-1; i++ {
		length += len(lines[i])
	}
	length += len(lines[rang.End.Line-1][:rang.End.Column-1])
	return length
}

// returns two new ranges, constructed by cutting innerRange out of wholeRange
// innerRange must be completely contained in wholeRange
// the resulting ranges are wholeRange.Start - innerRange.Start and innerRange.End - wholeRange.End
func cutRangeOut(wholeRange, innerRange token.Range) []token.Range {
	return []token.Range{
		{
			Start: wholeRange.Start,
			End:   innerRange.Start,
		},
		{
			Start: innerRange.End,
			End:   wholeRange.End,
		},
	}
}

// returns wether the given protocol.Position is inside rang
func isInRange(rang token.Range, pos protocol.Position) bool {
	if pos.Line < uint32(rang.Start.Line-1) || pos.Line > uint32(rang.End.Line-1) {
		return false
	}
	if pos.Line == uint32(rang.Start.Line-1) && pos.Line == uint32(rang.End.Line-1) {
		return pos.Character+1 >= uint32(rang.Start.Column-1) && pos.Character+1 <= uint32(rang.End.Column-1)
	}
	if pos.Line == uint32(rang.Start.Line-1) {
		return pos.Character+1 >= uint32(rang.Start.Column-1)
	}
	if pos.Line == uint32(rang.End.Line-1) {
		return pos.Character+1 <= uint32(rang.End.Column-1)
	}
	return true
}
