package main

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
	/*activeDocument = params.TextDocument.URI
	if err := parse(func(token.Token, string) {}); err != nil {
		log.Errorf("parser error: %s", err)
		return nil, err
	}*/

	items := make([]protocol.CompletionItem, 0)
	for _, s := range getDDPTypes() {
		items = append(items, protocol.CompletionItem{
			Kind:  Ptr(protocol.CompletionItemKindClass),
			Label: s,
		})
	}

	for _, s := range getDDPKeywords() {
		items = append(items, protocol.CompletionItem{
			Kind:  Ptr(protocol.CompletionItemKindKeyword),
			Label: s,
		})
	}

	return items, nil
}

func Ptr[T any](v T) *T {
	return &v
}

func getDDPTypes() []string {
	return []string{
		"Zahl",
		"Kommazahl",
		"Boolean",
		"Text",
		"Buchstabe",
		"Zahlen Liste",
		"Kommazahlen Liste",
		"Boolean Liste",
		"Text Liste",
		"Buchstaben Liste",
	}
}

func getDDPKeywords() []string {
	return []string{
		"pi",
		"e",
		"tau",
		"phi",
		"wahr",
		"falsch",
		"plus",
		"minus",
		"mal",
		"durch",
		"modulo",
		"hoch",
		"Wurzel",
		"Betrag",
		"und",
		"oder",
		"nicht",
		"gleich",
		"ungleich",
		"kleiner",
		"größer",
		"groesser",
		"ist",
		"der",
		"die",
		"von",
		"als",
		"wenn",
		"dann",
		"aber",
		"sonst",
		"solange",
		"für",
		"fuer",
		"jede",
		"jeden",
		"bis",
		"mit",
		"Schrittgröße",
		"Schrittgroesse",
		"Funktion",
		"Binde",
		"ein",
		"gib",
		"zurück",
		"zurueck",
		"nichts",
		"um",
		"Bit",
		"nach",
		"links",
		"rechts",
		"verschoben",
		"Größe",
		"Groesse",
		"Länge",
		"Laenge",
		"kontra",
		"logisch",
		"mache",
		"dem",
		"Parameter",
		"den",
		"Parametern",
		"vom",
		"Typ",
		"gibt",
		"eine",
		"einen",
		"macht",
		"kann",
		"so",
		"benutzt",
		"werden",
		"speichere",
		"das",
		"Ergebnis",
		"in",
		"verkettet",
		"addiere",
		"erhöhe",
		"erhoehe",
		"subtrahiere",
		"verringere",
		"multipliziere",
		"vervielfache",
		"dividiere",
		"teile",
		"verschiebe",
		"negiere",
		"an",
		"Stelle",
		"Logarithmus",
		"zur",
		"Basis",
		"definiert",
		"leere",
		"leeren",
		"aus",
		"besteht",
		"einer",
		"verlasse",
		"Mal",
		"Alias",
		"steht",
	}
}
