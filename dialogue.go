package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	// "archive/tar" FIXME: tar + gz then upload read only conf

	"gopkg.in/yaml.v2"
)

const (
	coveredci        = ".coveredci"
	localYML         = coveredci + ".yml"
	mimeJSON         = "application/json"
	mimeYAML         = "application/x-yaml"
	xAPIKeyHeader    = "X-Api-Key"
	xAuthTokenHeader = "X-Auth-Token"
)

type ymlCfg struct {
	AuthToken string
	Script    map[string][]string
}

func initDialogue(apiKey string) (*ymlCfg, aCmd) {
	yml := readYAML(localYML)

	// Has to be a string cause []byte gets base64-encoded
	fixtures := map[string]string{localYML: string(yml)}
	payload, err := json.Marshal(fixtures)
	if err != nil {
		log.Fatal(err)
	}
	cmdJSON, authToken := initPUT(apiKey, payload)
	cmd := unmarshalCmd(cmdJSON)

	var ymlConf struct {
		Start []string `yaml:"start"`
		Reset []string `yaml:"reset"`
		Stop  []string `yaml:"stop"`
	}
	if err := yaml.Unmarshal(yml, &ymlConf); err != nil {
		log.Fatal(err)
	}

	cfg := &ymlCfg{
		AuthToken: authToken,
		Script: map[string][]string{
			"start": ymlConf.Start,
			"reset": ymlConf.Reset,
			"stop":  ymlConf.Stop,
		},
	}
	return cfg, cmd
}

func next(cfg *ymlCfg, cmd aCmd) aCmd {
	if cmd.Kind() == "done" {
		return nil
	}

	rep := cmd.Exec(cfg)
	nextCmdJSON := nextPOST(cfg, rep)
	return unmarshalCmd(nextCmdJSON)
}

func readYAML(path string) []byte {
	fd, err := os.Open(path)
	if err != nil {
		log.Fatalf("You must provide a readable '.coveredci.yml' file in the current directory.\nError: %s\n", err)
	}
	defer fd.Close()

	yml, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Fatal("!yml: ", err)
	}

	return yml
}

func initPUT(apiKey string, JSON []byte) ([]byte, string) {
	var r *http.Request
	r, err := http.NewRequest(http.MethodPut, initURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Fatal("!initPUT: ", err)
	}

	r.Header.Set("Content-Type", mimeYAML)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set(xAPIKeyHeader, apiKey)
	client := &http.Client{}

	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Fatal("!PUT: ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("!read body: ", err)
	}
	log.Printf("🡱  %vμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, initURL, JSON, body)

	if resp.StatusCode != 201 {
		log.Fatal("!201: ", resp.Status)
	}

	authToken := resp.Header.Get(xAuthTokenHeader)
	if authToken == "" {
		log.Fatal("Could not acquire an AuthToken")
	}

	return body, authToken
}

func nextPOST(cfg *ymlCfg, payload []byte) []byte {
	r, err := http.NewRequest(http.MethodPost, nextURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal("!nextPOST: ", err)
	}

	r.Header.Set("content-type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set(xAuthTokenHeader, cfg.AuthToken)
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Fatal("!POST: ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("!read body: ", err)
	}
	log.Printf("🡱  %vμs POST %s\n  🡱  %s\n  🡳  %s\n", us, nextURL, payload, body)

	if resp.StatusCode != 200 {
		log.Fatal("!200: ", resp.Status)
	}

	return body
}
