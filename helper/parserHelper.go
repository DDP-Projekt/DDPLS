package helper

import (
	"github.com/DDP-Projekt/Kompilierer/src/token"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func GetAliasParamProtocolRange(aliasToken token.Token) protocol.Range {
	return ToProtocolRange(token.Range{
		Start: token.Position{
			Line:   aliasToken.Range.Start.Line,
			Column: aliasToken.Range.Start.Column + 2,
		},
		End: aliasToken.Range.End,
	})
}

func AliasParamNameEquals(t1 token.Token, name string) bool {
	return t1.Type == token.ALIAS_PARAMETER && t1.Literal == "<"+name+">"
}
