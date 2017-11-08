package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"fmt"

	"gopkg.in/yaml.v2"
)


func validateDocs(apiKey string, yml []byte) []byte {
	docs := struct {
		V uint `json:"v"`
		Blobs map[string]string `json:"blobs"`
	}{
		V: 1,
		Blobs: blobs(yml),
	}

	payload, err := json.Marshal(docs)
	if err != nil {
		log.Fatal(err)
	}

	return validationPOST(apiKey, payload)
}

func validationPOST(apiKey string, JSON []byte) []byte {
	r, err := http.NewRequest(http.MethodPost, docsURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Fatal(err)
	}

	r.Header.Set("Content-Type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set(xAPIKeyHeader, apiKey)
	client := &http.Client{}

	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("!read body: ", err)
	}
	log.Printf("🡱  %vμs POST %s\n  🡱  %s\n  🡳  %s\n", us, docsURL, JSON, body)

	if resp.StatusCode == 400 {
		reportValidationErrors(body)
		log.Fatal("Documentation validation failed")
	}

	if resp.StatusCode != 200 {
		log.Fatal("!200: ", resp.Status)
	}

	var validated struct {
		V uint `json:"v"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &validated); err != nil {
		log.Fatal(err)
	}
	if validated.Token == "" {
		log.Fatal("Could not acquire a validation token")
	}

	return body
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
		log.Fatal(err)
	}
	blobs[filePath] = string(fileData)

	return blobs
}
