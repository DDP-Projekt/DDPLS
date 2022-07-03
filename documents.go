package main

import "sync"

type DocumentState struct {
	Content string
	Uri     string
}

var documentStates = &sync.Map{}
var activeDocument string // uri of the active document

func addDocument(uri, content string) {
	documentStates.Store(uri, &DocumentState{
		Uri:     uri,
		Content: content,
	})
}

func getDocument(uri string) (*DocumentState, bool) {
	doc, ok := documentStates.Load(uri)
	return doc.(*DocumentState), ok
}

func deleteDocument(uri string) {
	documentStates.Delete(uri)
}
