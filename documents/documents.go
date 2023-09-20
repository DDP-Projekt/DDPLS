package documents

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
)

// represents the state of a single document
type DocumentState struct {
	Content     string      // the content of the document
	Uri         uri.URI     // the uri from the client
	Path        string      // the filepath as parsed from the uri
	Module      *ast.Module // the corresponding ddp module
	NeedReparse atomic.Bool // whether the document needs to be reparsed
	parseMutex  sync.Mutex  // the mutex used for parsing
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
	d.NeedReparse.Store(false)

	// cache all imports (like Duden modules etc.)
	if err == nil {
		for _, imprt := range d.Module.Imports {
			if imprt.Module != nil {
				imprt_uri := uri.FromPath(imprt.Module.FileName)
				if _, ok := modules[imprt_uri.Filepath()]; !ok {
					modules[imprt_uri.Filepath()] = imprt.Module
				}
			}
		}
	}

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
		if v != doc && v.Module != nil {
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
		if doc.NeedReparse.Load() {
			// check if doc is currently being reparsed by trying to aquire the mutex
			if doc.parseMutex.TryLock() {
				// it was not being parsed, so we unlock the mutex
				// which will be locked again by ReParse
				doc.parseMutex.Unlock()
				ok = dm.ReParse(doc.Uri, ddperror.EmptyHandler) == nil
			} else {
				// if it is being currently reparsed we wait for it to finish
				// by aquiring the mutex and then immediately unlock and return it
				doc.parseMutex.Lock()
				doc.parseMutex.Unlock()
			}
		}
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
