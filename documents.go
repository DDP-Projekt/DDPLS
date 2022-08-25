package main

import (
	"net/url"
	"strings"
	"sync"
)

type DocumentState struct {
	Content string
	Uri     string
	Path    string
}

var documentStates = &sync.Map{}
var activeDocument string // uri of the active document

func addDocument(uri, content string) {
	path, err := uriToPath(uri)
	if err != nil {
		log.Warningf("url.ParseRequestURI: %s", err)
	}
	documentStates.Store(uri, &DocumentState{
		Uri:     uri,
		Content: content,
		Path:    path,
	})
}

func getDocument(uri string) (*DocumentState, bool) {
	doc, ok := documentStates.Load(uri)
	return doc.(*DocumentState), ok
}

func deleteDocument(uri string) {
	documentStates.Delete(uri)
}

func uriToPath(uri string) (string, error) {
	parsed, err := url.ParseRequestURI(uri)
	path := uri
	if err != nil {
		return "", err
	} else {
		path = strings.TrimLeft(parsed.Path, "/")
	}
	return path, nil
}
