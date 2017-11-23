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
	r.Header.Set("User-Agent", binTitle)
	if apiKey != "" {
		r.Header.Set(xAPIKeyHeader, apiKey)
	}

	start := time.Now()
	resp, err := clientUtils.Do(r)
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

func maybeReportValidationErrors(errors []byte) error {
	if errors != nil {
		err := newDocsInvalidError(errors)
		log.Println("[ERR]", err)
		fmt.Println(err)
		return err
	}

	log.Println("[NFO] No validation errors found.")
	fmt.Println("No validation errors found.")
	//TODO: make it easy to use returned token
	return nil
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
