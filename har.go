package main

import (
	"log"
	"net/http"

	"gopkg.in/cardigann/harhar.v0"
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
	harData := harCollector.HAR
	if harData == nil {
		log.Fatal("[ERR] HAR is nil!")
	}
	return harData
}

func clearHAR() {
	harCollector = nil
	clientReq = nil
}
