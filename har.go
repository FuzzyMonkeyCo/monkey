package main

import (
	"net/http"

	"gopkg.in/CoveredCI/harhar.v0"
)

var (
	clientReq    *http.Client
	harCollector *harhar.Recorder
)

type har *harhar.Log

func newHARTransport() {
	harCollector = harhar.NewRecorder()
	clientReq = &http.Client{Transport: harCollector}
}

func isHARReady() bool {
	return clientReq != nil
}

func lastHAR() har {
	last := harhar.NewHAR()
	all := harCollector.HAR.Log.Entries
	lastEntry := all[len(all)-1]
	//FIXME: even less data actually needs to be sent
	last.Log.Entries = []harhar.Entry{lastEntry}
	return &last.Log
}

func clearHAR() {
	harCollector = nil
	clientReq = nil
}
