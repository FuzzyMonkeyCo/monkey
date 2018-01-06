package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-yaml/yaml"
)

const localYML = ".fuzzymonkey.yml"

func validateDocs(apiKey string, yml []byte) (rep []byte, err error) {
	blobs, err := makeBlobs(yml)
	if err != nil {
		return
	}

	docs := struct {
		V     uint              `json:"v"`
		Blobs map[string]string `json:"blobs"`
	}{
		V:     1,
		Blobs: blobs,
	}

	payload, err := json.Marshal(docs)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	rep, err = validationReq(apiKey, payload)
	return
}

func validationReq(apiKey string, JSON []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPut, docsURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set("Content-Type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binTitle)
	if apiKey != "" {
		r.Header.Set(xAPIKeyHeader, apiKey)
	}

	start := time.Now()
	resp, err := clientUtils.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if rep, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG] 🡱  %dμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, docsURL, JSON, rep)

	if resp.StatusCode == 400 {
		err = newDocsInvalidError(rep)
		log.Println("[ERR]", err)
		fmt.Println(err)
		return
	}

	if resp.StatusCode != 201 {
		err = newStatusError(201, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	var validated struct {
		V     uint   `json:"v"`
		Token string `json:"token"`
	}
	if err = json.Unmarshal(rep, &validated); err != nil {
		log.Println("[ERR]", err)
		return
	}

	if validated.Token == "" {
		err = fmt.Errorf("Could not acquire a validation token")
		log.Println("[ERR]", err)
		fmt.Println(err)
		return
	}

	log.Println("[NFO] No validation errors found.")
	fmt.Println("No validation errors found.")
	//FIXME: auto-reuse returned token
	return
}

func makeBlobs(yml []byte) (blobs map[string]string, err error) {
	blobs = map[string]string{localYML: string(yml)}

	var ymlConfPartial struct {
		Doc struct {
			File string `yaml:"file"`
		} `yaml:"documentation"`
	}
	if err = yaml.Unmarshal(yml, &ymlConfPartial); err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Failed to parse %s: %+v\n", localYML, err)
		return
	}

	//FIXME: force relative paths & nested under workdir. Watch out for links
	filePath := ymlConfPartial.Doc.File
	if "" == filePath {
		return
	}

	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Could not read '%s'\n", filePath)
		return
	}
	blobs[filePath] = string(fileData)

	return
}
