package documents

import (
	"sync"

	"github.com/DDP-Projekt/DDPLS/uri"
)

type DocumentState struct {
	Content string
	Uri     uri.URI
	Path    string
}

// all the documents state
// keys are the params.TextDocument.URI
var documentStates = &sync.Map{}
var Active string // uri key of the active document

func Add(vscURI, content string) {
	docURI := uri.FromURI(vscURI)
	documentStates.Store(vscURI, &DocumentState{
		Uri:     docURI,
		Content: content,
		Path:    docURI.Filepath(),
	})
}

func Get(docURI string) (*DocumentState, bool) {
	doc, ok := documentStates.Load(docURI)
	return doc.(*DocumentState), ok
}

func Delete(docURI string) {
	documentStates.Delete(docURI)
}
