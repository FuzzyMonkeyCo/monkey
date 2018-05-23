package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func doAuth(cfg *ymlCfg, apiKey string, N uint) (err error) {
	if apiKey == "" {
		err = fmt.Errorf("$%s is unset", envAPIKey)
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	type options struct {
		Tests uint   `json:"num_tests"`
		Seed  string `json:"seed",omitempty`
	}

	payload := struct {
		V       uint    `json:"v"`
		YML     uint    `json:"version"`
		Client  string  `json:"client"`
		Options options `json:"options"`
	}{
		V:      v,
		YML:    cfg.Version,
		Client: binTitle,
		Options: options{
			Tests: N,
		},
	}

	buf := &bytes.Buffer{}
	if err = json.NewEncoder(buf).Encode(&payload); err != nil {
		log.Println("[ERR]", err)
		return
	}

	var URL string
	if binVersion == "0.0.0" {
		URL = "http://auth.dev.fuzzymonkey.co/1/token"
	} else {
		//FIXME: use HTTPS
		URL = "http://auth.fuzzymonkey.co/1/token"
	}

	r, err := http.NewRequest(http.MethodPut, URL, buf)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set(headerContentType, mimeJSON)
	r.Header.Set(headerAccept, mimeJSON)
	r.Header.Set(headerUserAgent, binTitle)
	r.Header.Set(headerXAPIKey, apiKey)

	log.Printf("[DBG] 🡱  PUT %s\n  🡱  %#v\n", URL, payload)
	start := time.Now()
	resp, err := clientUtils.Do(r)
	log.Printf("[DBG] ❙ %dμs\n", time.Since(start)/time.Microsecond)
	if err != nil {
		// if here probably a HomeConnectionError
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err = newStatusError(http.StatusCreated, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	authToken := resp.Header.Get(headerXAuthToken)
	if authToken == "" {
		err = errors.New("Could not acquire an AuthToken")
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	log.Printf("[NFO] got auth token: %s\n", authToken)
	cfg.AuthToken = authToken
	return
}
