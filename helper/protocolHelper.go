package helper

import (
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/DDPLS/documents"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// converts a token.Range to a protocol.Range
func ToProtocolRange(rang token.Range) protocol.Range {
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

func FromProtocolRange(rang protocol.Range) token.Range {
	return token.Range{
		Start: token.Position{
			Line:   uint(rang.Start.Line + 1),
			Column: uint(rang.Start.Character + 1),
		},
		End: token.Position{
			Line:   uint(rang.End.Line + 1),
			Column: uint(rang.End.Character + 1),
		},
	}
}

func ToProtocolPosition(pos token.Position) protocol.Position {
	return protocol.Position{
		Line:      uint32(pos.Line - 1),
		Character: uint32(pos.Column - 1),
	}
}

func FromProtocolPosition(pos protocol.Position) token.Position {
	return token.Position{
		Line:   uint(pos.Line + 1),
		Column: uint(pos.Character + 1),
	}
}

// returns the length of a token.Range
func GetRangeLength(rang token.Range, doc *documents.DocumentState) int {
	if rang.Start.Line == rang.End.Line {
		return int(rang.End.Column - rang.Start.Column)
	}
	lines := strings.Split(doc.Content, "\n")
	length := utf8.RuneCountInString(lines[rang.Start.Line-1][rang.Start.Column-1:])
	for i := rang.Start.Line; i < rang.End.Line-1; i++ {
		length += utf8.RuneCountInString(lines[i])
	}
	length += utf8.RuneCountInString(lines[rang.End.Line-1][:rang.End.Column-1])
	return length
}

// returns two new ranges, constructed by cutting innerRange out of wholeRange
// innerRange must be completely contained in wholeRange
// the resulting ranges are wholeRange.Start - innerRange.Start and innerRange.End - wholeRange.End
func CutRangeOut(wholeRange, innerRange token.Range) []token.Range {
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
func IsInRange(rang token.Range, pos protocol.Position) bool {
	if pos.Line < uint32(rang.Start.Line-1) || pos.Line > uint32(rang.End.Line-1) {
		return false
	}
	if pos.Line == uint32(rang.Start.Line-1) && pos.Line == uint32(rang.End.Line-1) {
		return pos.Character >= uint32(rang.Start.Column-1) && pos.Character <= uint32(rang.End.Column-1)
	}
	if pos.Line == uint32(rang.Start.Line-1) {
		return pos.Character >= uint32(rang.Start.Column-1)
	}
	if pos.Line == uint32(rang.End.Line-1) {
		return pos.Character <= uint32(rang.End.Column-1)
	}
	return true
}

func Contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
