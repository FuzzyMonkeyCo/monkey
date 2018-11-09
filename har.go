package main

import (
	"net/http"

	"github.com/cardigann/harhar"
)

var (
	clientReq    *http.Client
	harCollector *harhar.Recorder
)

type harEntry *harhar.Entry

func newHARTransport() {
	harCollector = harhar.NewRecorder()
	clientReq = &http.Client{Transport: harCollector}
}

func isHARReady() bool {
	return clientReq != nil
}

func lastHAR() harEntry {
	all := harCollector.HAR.Log.Entries
	//FIXME: even less data actually needs to be sent
	entry := all[len(all)-1]
	// entry.Request = nil
	return &entry
}

func clearHAR() {
	harCollector = nil
	clientReq = nil
}
