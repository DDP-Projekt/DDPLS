package documents

import (
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/DDP-Projekt/DDPLS/uri"
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/parser"
)

var preparsed_duden map[string]*ast.Module

func init() {
	preparsed_duden = make(map[string]*ast.Module)
	filepath.WalkDir(ddppath.Duden, func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".ddp" {
			return nil
		}
		mod, err := parser.Parse(parser.Options{
			FileName: path,
			Modules:  preparsed_duden,
		})
		if err != nil {
			return err
		}
		preparsed_duden[mod.FileName] = mod
		return nil
	})
}

// represents the state of a single document
type DocumentState struct {
	Content      string           // the content of the document
	Uri          uri.URI          // the uri from the client
	Path         string           // the filepath as parsed from the uri
	Module       *ast.Module      // the corresponding ddp module
	NeedReparse  atomic.Bool      // whether the document needs to be reparsed
	LatestErrors []ddperror.Error // the errors from the last parsing
}

func (d *DocumentState) newErrorCollector() ddperror.Handler {
	d.LatestErrors = make([]ddperror.Error, 0, 10)
	return func(err ddperror.Error) {
		d.LatestErrors = append(d.LatestErrors, err)
	}
}

var n = 0

func (d *DocumentState) reParseInContext(modules map[string]*ast.Module, errorHandler ddperror.Handler) (err error) {
	if duden_mod, ok := preparsed_duden[d.Path]; ok {
		d.Module = duden_mod
	} else {
		// clear generic instantiations to not leak memory
		ast.VisitModule(d.Module, &genericsClearer{mod: d.Module})

		duden := make(map[string]*ast.Module, len(preparsed_duden))
		maps.Copy(duden, preparsed_duden)
		d.Module, err = parser.Parse(parser.Options{
			FileName: d.Path,
			Source:   []byte(d.Content),
			// TODO: make this work better
			// Modules:      merge_map_into(preparsed_duden, modules),
			Modules:      duden,
			ErrorHandler: errorHandler,
		})
	}

	d.NeedReparse.Store(false)

	return err
}

// a synced map that manages document states
type DocumentManager struct {
	mu             sync.Mutex
	documentStates map[uri.URI]*DocumentState
}

func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		mu:             sync.Mutex{},
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
	defer dm.mu.Unlock()
	dm.documentStates[docURI] = docState

	return dm.reParse(docURI, docState.newErrorCollector())
}

func (dm *DocumentManager) reParse(docUri uri.URI, errHndl ddperror.Handler) error {
	doc, ok := dm.documentStates[docUri]
	if !ok {
		return fmt.Errorf("document %s not found in map", docUri)
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
	dm.mu.Lock()
	defer dm.mu.Unlock()
	doc, ok := dm.documentStates[uri.FromURI(vscURI)]
	if !ok {
		return nil, ok
	}

	if doc.NeedReparse.Load() {
		ok = dm.reParse(doc.Uri, doc.newErrorCollector()) == nil
	}
	return doc, ok
}

func (dm *DocumentManager) GetFromMod(mod *ast.Module) (*DocumentState, bool) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
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

// merges a into b and returns b
func merge_map_into[K comparable, V any](a, b map[K]V) map[K]V {
	for k, v := range a {
		b[k] = v
	}
	return b
}
