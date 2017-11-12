package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	localYML         = ".coveredci.yml"
	mimeJSON         = "application/json"
	mimeYAML         = "application/x-yaml"
	xAPIKeyHeader    = "X-Api-Key"
	xAuthTokenHeader = "X-Auth-Token"
)

type ymlCfg struct {
	AuthToken string
	Host      string
	Port      string
	Script    map[string][]string
	FinalHost string
	FinalPort string
}

func initDialogue(apiKey string) (*ymlCfg, aCmd) {
	yml := readYAML(localYML)

	validationJSON, errors := validateDocs(apiKey, yml)
	if errors != nil {
		reportValidationErrors(errors)
		log.Fatal("Documentation validation failed")
	}

	cmdJSON, authToken := initPUT(apiKey, validationJSON)
	cmd := unmarshalCmd(cmdJSON)

	var ymlConf struct {
		Start []string `yaml:"start"`
		Reset []string `yaml:"reset"`
		Stop  []string `yaml:"stop"`
		Doc   struct {
			Host string `yaml:"host"`
			Port string `yaml:"port"`
		} `yaml:"documentation"`
	}
	if err := yaml.Unmarshal(yml, &ymlConf); err != nil {
		log.Fatal("[ERR] ", err)
	}

	cfg := &ymlCfg{
		AuthToken: authToken,
		Host:      ymlConf.Doc.Host,
		Port:      ymlConf.Doc.Port,
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

	// Sometimes sets cfg.Final* fields
	rep := cmd.Exec(cfg)

	nextCmdJSON := nextPOST(cfg, rep)
	return unmarshalCmd(nextCmdJSON)
}

func readYAML(path string) []byte {
	fd, err := os.Open(path)
	if err != nil {
		log.Println("You must provide a readable '.coveredci.yml' file in the current directory.")
		log.Fatalf("Error: %s\n", err)
	}
	defer fd.Close()

	yml, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Fatal("[ERR] !yml: ", err)
	}

	return yml
}

func initPUT(apiKey string, JSON []byte) ([]byte, string) {
	r, err := http.NewRequest(http.MethodPut, initURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	r.Header.Set("Content-Type", mimeYAML)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binVersion)
	r.Header.Set(xAPIKeyHeader, apiKey)
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
	log.Printf("[DBG] 🡱  %vμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, initURL, JSON, body)

	if resp.StatusCode != 201 {
		log.Fatal("[ERR] !201: ", resp.Status)
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
		log.Fatal("[ERR] ", err)
	}

	r.Header.Set("content-type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binVersion)
	r.Header.Set(xAuthTokenHeader, cfg.AuthToken)
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
	log.Printf("[DBG] 🡱  %vμs POST %s\n  🡱  %s\n  🡳  %s\n", us, nextURL, payload, body)

	if resp.StatusCode != 200 {
		log.Fatal("[ERR] !200: ", resp.Status)
	}

	return body
}
