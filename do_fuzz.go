package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"
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

func newFuzz(cfg *YmlCfg, spec *SpecIR) (cmd someCmd, err error) {
	initer := &FuzzCfg{
		Config: cfg,
		Spec:   spec,
	}
	payload, err := proto.Marshal(initer)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if err = maybePreStart(cfg); err != nil {
		return
	}

	cmdJSON, err := fuzzNew(cfg, payload)
	if err != nil {
		return
	}

	cmd, err = unmarshalCmd(cmdJSON)
	return
}

func fuzzNext(cfg *YmlCfg, cmd someCmd) (someCmd someCmd, err error) {
	// Sometimes sets cfg.Runtime.Final* fields
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

func fuzzNew(cfg *YmlCfg, payload []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPut, apiFuzzNew, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set(headerContentType, mimeYAML)
	r.Header.Set(headerAccept, mimeJSON)
	r.Header.Set(headerUserAgent, binTitle)
	r.Header.Set(headerXAuthToken, cfg.AuthToken)

	log.Printf("[DBG] 🡱  PUT %s\n  🡱  %dB\n", apiFuzzNew, len(payload))
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

func nextPOST(cfg *YmlCfg, payload []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPost, apiFuzzNext, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set(headerContentType, mimeJSON)
	r.Header.Set(headerAccept, mimeJSON)
	r.Header.Set(headerUserAgent, binTitle)
	r.Header.Set(headerXAuthToken, cfg.AuthToken)

	log.Printf("[DBG] 🡱  POST %s\n  🡱  %s\n", apiFuzzNext, payload)
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
