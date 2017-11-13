package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v2"
)

func validateDocs(apiKey string, yml []byte) ([]byte, []byte) {
	docs := struct {
		V     uint              `json:"v"`
		Blobs map[string]string `json:"blobs"`
	}{
		V:     1,
		Blobs: blobs(yml),
	}

	payload, err := json.Marshal(docs)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	return validationReq(apiKey, payload)
}

func validationReq(apiKey string, JSON []byte) ([]byte, []byte) {
	r, err := http.NewRequest(http.MethodPut, docsURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	r.Header.Set("Content-Type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set("User-Agent", binVersion)
	if apiKey != "" {
		r.Header.Set(xAPIKeyHeader, apiKey)
	}
	client := &http.Client{}

	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("[ERR] !read body: ", err)
	}
	log.Printf("[DBG] 🡱  %vμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, docsURL, JSON, body)

	if resp.StatusCode == 400 {
		return nil, body
	}

	if resp.StatusCode != 201 {
		log.Fatal("[ERR] !201: ", resp.Status)
	}

	var validated struct {
		V     uint   `json:"v"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &validated); err != nil {
		log.Fatal("[ERR] ", err)
	}
	if validated.Token == "" {
		log.Fatal("Could not acquire a validation token")
	}

	return body, nil
}

func reportValidationErrors(errors []byte) {
	fmt.Println("Validation errors:")

	var out bytes.Buffer
	err := json.Indent(&out, errors, "", "  ")
	if err != nil {
		fmt.Println(string(errors))
	}

	fmt.Println(out.String())
}

func blobs(yml []byte) map[string]string {
	blobs := map[string]string{localYML: string(yml)}

	var ymlConfPartial struct {
		Doc struct {
			File string `yaml:"file"`
		} `yaml:"documentation"`
	}
	// YML is not yet validated, so ignore errors
	yaml.Unmarshal(yml, &ymlConfPartial)

	filePath := ymlConfPartial.Doc.File
	if "" == filePath {
		return blobs
	}

	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("[ERR]", err)
		log.Fatal("Could not read ", filePath)
	}
	blobs[filePath] = string(fileData)

	return blobs
}
