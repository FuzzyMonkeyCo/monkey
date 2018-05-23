package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	v                 = 1
	mimeJSON          = "application/json"
	mimeYAML          = "application/x-yaml"
	headerContentType = "Content-Type"
	headerAccept      = "Accept"
	headerUserAgent   = "User-Agent"
	headerXAPIKey     = "X-Api-Key"
	headerXAuthToken  = "X-Auth-Token"
)

var (
	apiFuzz  string
	fuzzNew  string
	fuzzNext string
)

func newFuzz(cfg *ymlCfg, spec []byte) (cmd someCmd, err error) {
	blobs, err := makeBlobs(cfg, spec)
	if err != nil {
		return
	}

	if err = maybePreStart(cfg); err != nil {
		return
	}

	if binVersion == "0.0.0" {
		apiFuzz = "http://fuzz.dev.fuzzymonkey.co/1"
	} else {
		//FIXME: use HTTPS
		apiFuzz = "http://fuzz.fuzzymonkey.co/1"
	}
	fuzzNew = apiFuzz + "/new"
	fuzzNext = apiFuzz + "/next"

	cmdJSON, err := initPUT(cfg, blobs)
	if err != nil {
		return
	}

	cmd, err = unmarshalCmd(cmdJSON)
	return
}

func next(cfg *ymlCfg, cmd someCmd) (someCmd someCmd, err error) {
	// Sometimes sets cfg.Final* fields
	rep, err := cmd.Exec(cfg)
	if err != nil {
		return
	}

	nextCmdJSON, err := nextPOST(cfg, rep)
	if err != nil {
		return
	}

	someCmd, err = unmarshalCmd(nextCmdJSON)
	return
}

func initPUT(cfg *ymlCfg, JSON []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPut, fuzzNew, bytes.NewBuffer(JSON))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set("Content-Type", mimeYAML)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binTitle)
	r.Header.Set(headerXAuthToken, cfg.AuthToken)

	log.Printf("[DBG] 🡱  PUT %s\n  🡱  %s\n", fuzzNew, JSON)
	start := time.Now()
	resp, err := clientUtils.Do(r)
	log.Printf("[DBG] ❙ %dμs\n", time.Since(start)/time.Microsecond)
	if err != nil {
		// if here probably a HomeConnectionError
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if rep, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG]\n  🡳  %s\n", rep)

	if resp.StatusCode != http.StatusCreated {
		err = newStatusError(http.StatusCreated, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	cfg.AuthToken = resp.Header.Get(headerXAuthToken)
	return
}

func nextPOST(cfg *ymlCfg, payload []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPost, fuzzNext, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set("content-type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binTitle)
	r.Header.Set(headerXAuthToken, cfg.AuthToken)

	log.Printf("[DBG] 🡱  POST %s\n  🡱  %s\n", fuzzNext, payload)
	start := time.Now()
	resp, err := clientUtils.Do(r)
	log.Printf("[DBG] ❙ %dμs\n", time.Since(start)/time.Microsecond)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if rep, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG]\n  🡳  %s\n", rep)

	if resp.StatusCode != http.StatusOK {
		err = newStatusError(http.StatusOK, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	cfg.AuthToken = resp.Header.Get(headerXAuthToken)
	return
}

func makeBlobs(cfg *ymlCfg, spec []byte) (payload []byte, err error) {
	blobs := map[string]string{localYML: cfg.Kind}
	blobs[cfg.File] = string(spec)

	docs := struct {
		V     uint              `json:"v"`
		Blobs map[string]string `json:"blobs"`
	}{
		V:     v,
		Blobs: blobs,
	}
	if payload, err = json.Marshal(docs); err != nil {
		log.Println("[ERR]", err)
	}

	return
}
