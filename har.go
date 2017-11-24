package main

import (
	"net/http"

	"gopkg.in/CoveredCI/harhar.v0"
)

var (
	clientReq    *http.Client
	harCollector *harhar.Recorder
)

type har *harhar.HAR

func newHARTransport() {
	harCollector = harhar.NewRecorder()
	clientReq = &http.Client{Transport: harCollector}
}

func isHARReady() bool {
	return clientReq != nil
}

func readHAR() har {
	return harCollector.HAR
}

func clearHAR() {
	harCollector = nil
	clientReq = nil
}
