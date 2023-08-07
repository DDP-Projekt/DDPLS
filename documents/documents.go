package documents

import (
	"fmt"
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

func (d *DocumentState) reParseInContext(modules map[string]*ast.Module, errorHandler ddperror.Handler) (err error) {
	d.parseMutex.Lock()
	defer d.parseMutex.Unlock()

	d.Module, err = parser.Parse(parser.Options{
		FileName:     d.Path,
		Source:       []byte(d.Content),
		Modules:      modules,
		ErrorHandler: errorHandler,
	})

	return err
}

// a synced map that manages document states
type DocumentManager struct {
	mu             sync.RWMutex
	documentStates map[uri.URI]*DocumentState
}

func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		mu:             sync.RWMutex{},
		documentStates: make(map[uri.URI]*DocumentState),
	}
}

// adds a document to the map
// and parses its content
func (dm *DocumentManager) AddAndParse(vscURI, content string) error {
	docURI := uri.FromURI(vscURI)
	docState := &DocumentState{
		Uri:     docURI,
		Content: content,
		Path:    docURI.Filepath(),
	}
	dm.mu.Lock()
	dm.documentStates[docURI] = docState
	dm.mu.Unlock()
	return dm.ReParse(docURI, ddperror.EmptyHandler)
}

func (dm *DocumentManager) ReParse(docUri uri.URI, errHndl ddperror.Handler) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	doc, ok := dm.documentStates[docUri]
	if !ok {
		return fmt.Errorf("Document %s not found in map", docUri)
	}

	modules := map[string]*ast.Module{}
	for _, v := range dm.documentStates {
		if v != doc {
			modules[v.Module.FileName] = v.Module
		}
	}

	return doc.reParseInContext(modules, errHndl)
}

func (dm *DocumentManager) Get(vscURI string) (*DocumentState, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	doc, ok := dm.documentStates[uri.FromURI(vscURI)]
	if ok {
		return doc, ok
	} else {
		return nil, ok
	}
}

func (dm *DocumentManager) GetFromMod(mod *ast.Module) (*DocumentState, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	for _, v := range dm.documentStates {
		if v.Module == mod {
			return v, true
		}
	}
	return nil, false
}

func (dm *DocumentManager) Delete(vscURI string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	delete(dm.documentStates, uri.FromURI(vscURI))
}
