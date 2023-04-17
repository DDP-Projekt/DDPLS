package documents

import (
	"sync"

	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
)

// represents the state of a single document
type DocumentState struct {
	Content    string      // the content of the document
	Uri        uri.URI     // the uri from the client
	Path       string      // the filepath as parsed from the uri
	Module     *ast.Module // the corresponding ddp module
	parseMutex sync.Mutex  // the mutex used for parsing
}

func (d *DocumentState) ReParse(errorHandler ddperror.Handler) (err error) {
	d.parseMutex.Lock()
	defer d.parseMutex.Unlock()

	modules := map[string]*ast.Module{}
	documentStates.Range(func(_, value any) bool {
		doc := value.(*DocumentState)
		if doc != d {
			modules[doc.Module.FileName] = doc.Module
		}
		return true
	})

	d.Module, err = parser.Parse(parser.Options{
		FileName:     d.Path,
		Source:       []byte(d.Content),
		Modules:      modules,
		ErrorHandler: errorHandler,
	})

	return err
}

// all the documents state
// keys are the params.TextDocument.URI
var documentStates = &sync.Map{}

// adds a document to the map
// and parses its content
func AddAndParse(vscURI, content string) error {
	docURI := uri.FromURI(vscURI)
	docState := &DocumentState{
		Uri:     docURI,
		Content: content,
		Path:    docURI.Filepath(),
	}
	documentStates.Store(vscURI, docState)
	return docState.ReParse(ddperror.EmptyHandler)
}

func Get(docURI string) (*DocumentState, bool) {
	doc, ok := documentStates.Load(docURI)
	if ok {
		return doc.(*DocumentState), ok
	} else {
		return nil, ok
	}
}

func Delete(docURI string) {
	documentStates.Delete(docURI)
}
