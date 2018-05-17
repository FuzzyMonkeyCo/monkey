package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	v                = 1
	mimeJSON         = "application/json"
	mimeYAML         = "application/x-yaml"
	xAPIKeyHeader    = "X-Api-Key"
	xAuthTokenHeader = "X-Auth-Token"
)

var (
	apiFuzz  string
	fuzzNew  string
	fuzzNext string
)

func newFuzz(cfg *ymlCfg, apiKey string, spec []byte) (cmd someCmd, err error) {
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

	cmdJSON, authToken, err := initPUT(apiKey, blobs)
	if err != nil {
		return
	}
	log.Printf("[NFO] got auth token: %s\n", authToken)
	cfg.AuthToken = authToken

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

func initPUT(apiKey string, JSON []byte) (rep []byte, authToken string, err error) {
	r, err := http.NewRequest(http.MethodPut, fuzzNew, bytes.NewBuffer(JSON))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set("Content-Type", mimeYAML)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binTitle)
	r.Header.Set(xAPIKeyHeader, apiKey)

	log.Printf("[DBG] 🡱  PUT %s\n  🡱  %s\n", fuzzNew, JSON)
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

	if resp.StatusCode != 201 {
		err = newStatusError(201, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	authToken = resp.Header.Get(xAuthTokenHeader)
	if authToken == "" {
		err = fmt.Errorf("Could not acquire an AuthToken")
		log.Println("[ERR]", err)
		fmt.Println(err)
	}
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
	r.Header.Set(xAuthTokenHeader, cfg.AuthToken)

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

	if resp.StatusCode != 200 {
		err = newStatusError(200, resp.Status)
		log.Println("[ERR]", err)
	}
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
