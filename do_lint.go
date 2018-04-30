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

	"github.com/googleapis/gnostic/OpenAPIv3"
	"github.com/googleapis/gnostic/compiler"
	"strings"
)

const localYML = ".fuzzymonkey.yml"

func lintDocs(apiKey string, yml []byte) (rep []byte, err error) {
	docPath, err := findBlobs(yml)
	if err != nil {
		return
	}
	fmt.Println("Documentation is at", docPath)

	docBytes, err := ioutil.ReadFile(docPath)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	info, err := compiler.ReadInfoFromBytes(docPath, docBytes)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	infoMap, ok := compiler.UnpackMap(info)
	if !ok {
		err = fmt.Errorf("format:unknown")
		log.Println("[ERR]", err)
		return
	}
	openapi, ok := compiler.MapValueForKey(infoMap, "openapi").(string)
	if !ok || !strings.HasPrefix(openapi, "3.0") {
		err = fmt.Errorf("format:unsupported")
		log.Println("[ERR]", err)
		return
	}

	doc, err := openapi_v3.NewDocument(info, compiler.NewContext("$root", nil))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	rawInfo, ok := doc.ToRawInfo().(yaml.MapSlice)
	if !ok || rawInfo == nil {
		err = fmt.Errorf("!yaml")
		log.Println("[ERR]", err)
		return
	}
	bytes, err := yaml.Marshal(rawInfo)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	fmt.Println(bytes)

	_, err = doc.ResolveReferences(docPath)

	rawInfo, ok = doc.ToRawInfo().(yaml.MapSlice)
	if !ok || rawInfo == nil {
		err = fmt.Errorf("!yaml")
		log.Println("[ERR]", err)
		return
	}
	if bytes, err = yaml.Marshal(rawInfo); err != nil {
		log.Println("[ERR]", err)
		return
	}
	fmt.Println(bytes)

	return ///

	blobs, err := makeBlobs(yml, docPath)
	if err != nil {
		return
	}

	docs := struct {
		V     uint              `json:"v"`
		Blobs map[string]string `json:"blobs"`
	}{
		V:     v,
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
	r, err := http.NewRequest(http.MethodPut, lintURL, bytes.NewBuffer(JSON))
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

	log.Printf("[DBG] 🡱  PUT %s\n  🡱  %s\n", lintURL, JSON)
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

	if validated.V != v || validated.Token == "" {
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

func findBlobs(yml []byte) (path string, err error) {
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
	path = ymlConfPartial.Doc.File
	if "" == path {
		err = fmt.Errorf("Path to documentation is empty")
		log.Println("[ERR]", err)
		return
	}
	return
}

func makeBlobs(yml []byte, filePath string) (blobs map[string]string, err error) {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Could not read '%s'\n", filePath)
		return
	}
	blobs = map[string]string{localYML: string(yml)}
	blobs[filePath] = string(fileData)

	return
}
