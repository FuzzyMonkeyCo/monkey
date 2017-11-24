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

func validateDocs(apiKey string, yml []byte) ([]byte, error) {
	blobs, err := makeBlobs(yml)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return validationReq(apiKey, payload)
}

func validationReq(apiKey string, JSON []byte) ([]byte, error) {
	r, err := http.NewRequest(http.MethodPut, docsURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	r.Header.Set("Content-Type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set("User-Agent", binTitle)
	if apiKey != "" {
		r.Header.Set(xAPIKeyHeader, apiKey)
	}

	start := time.Now()
	resp, err := clientUtils.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	log.Printf("[DBG] 🡱  %vμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, docsURL, JSON, body)

	if resp.StatusCode == 400 {
		err := newDocsInvalidError(body)
		log.Println("[ERR]", err)
		fmt.Println(err)
		return nil, err
	}

	if resp.StatusCode != 201 {
		err := fmt.Errorf("not 201: %v", resp.Status)
		log.Println("[ERR]", err)
		return nil, err
	}

	var validated struct {
		V     uint   `json:"v"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &validated); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	if validated.Token == "" {
		err := fmt.Errorf("Could not acquire a validation token")
		log.Println("[ERR]", err)
		fmt.Println(err)
		return nil, err
	}

	log.Println("[NFO] No validation errors found.")
	fmt.Println("No validation errors found.")
	//TODO: make it easy to use returned token
	return body, nil
}

type docsInvalidError struct {
	Errors string
}

func (e *docsInvalidError) Error() string {
	return e.Errors
}

func newDocsInvalidError(errors []byte) *docsInvalidError {
	start, end := "Validation errors:", "Documentation validation failed."
	var theErrors string
	var out bytes.Buffer
	err := json.Indent(&out, errors, "", "  ")
	if err != nil {
		theErrors = string(errors)
	}
	theErrors = out.String()

	return &docsInvalidError{start + "\n" + theErrors + "\n" + end}
}

func makeBlobs(yml []byte) (map[string]string, error) {
	blobs := map[string]string{localYML: string(yml)}

	var ymlConfPartial struct {
		Doc struct {
			File string `yaml:"file"`
		} `yaml:"documentation"`
	}
	if err := yaml.Unmarshal(yml, &ymlConfPartial); err != nil {
		return blobs, nil
	}

	filePath := ymlConfPartial.Doc.File
	if "" == filePath {
		return blobs, nil
	}

	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Could not read '%s'\n", filePath)
		return nil, err
	}
	blobs[filePath] = string(fileData)

	return blobs, nil
}
