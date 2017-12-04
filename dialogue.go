package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const (
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

func initDialogue(apiKey string) (cfg *ymlCfg, cmd aCmd, err error) {
	yml, err := readYML()
	if err != nil {
		return
	}

	validationJSON, err := validateDocs(apiKey, yml)
	if err != nil {
		return
	}

	cmdJSON, authToken, err := initPUT(apiKey, validationJSON)
	if err != nil {
		return
	}
	cmd, err = unmarshalCmd(cmdJSON)
	if err != nil {
		return
	}

	var ymlConf struct {
		Start []string `yaml:"start"`
		Reset []string `yaml:"reset"`
		Stop  []string `yaml:"stop"`
		Doc   struct {
			Host string `yaml:"host"`
			Port string `yaml:"port"`
		} `yaml:"documentation"`
	}
	if err = yaml.Unmarshal(yml, &ymlConf); err != nil {
		log.Println("[ERR]", err)
		return
	}

	cfg = &ymlCfg{
		AuthToken: authToken,
		Host:      ymlConf.Doc.Host,
		Port:      ymlConf.Doc.Port,
		Script: map[string][]string{
			"start": ymlConf.Start,
			"reset": ymlConf.Reset,
			"stop":  ymlConf.Stop,
		},
	}
	return
}

func next(cfg *ymlCfg, cmd aCmd) (someCmd aCmd, err error) {
	// Sometimes sets cfg.Final* fields
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

func readYML() (yml []byte, err error) {
	fd, err := os.Open(localYML)
	if err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("You must provide a readable %s file in the current directory.\n", localYML)
		return
	}
	defer fd.Close()

	yml, err = ioutil.ReadAll(fd)
	if err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func initPUT(apiKey string, JSON []byte) (rep []byte, authToken string, err error) {
	r, err := http.NewRequest(http.MethodPut, initURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set("Content-Type", mimeYAML)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binTitle)
	r.Header.Set(xAPIKeyHeader, apiKey)

	start := time.Now()
	resp, err := clientUtils.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	rep, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG] 🡱  %dμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, initURL, JSON, rep)

	if resp.StatusCode != 201 {
		err = newStatusError(201, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	authToken = resp.Header.Get(xAuthTokenHeader)
	if authToken == "" {
		err = fmt.Errorf("Could not acquire an AuthToken")
		log.Println("[ERR]", err)
		fmt.Println(err)
	}
	return
}

func nextPOST(cfg *ymlCfg, payload []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPost, nextURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set("content-type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
	r.Header.Set("User-Agent", binTitle)
	r.Header.Set(xAuthTokenHeader, cfg.AuthToken)

	start := time.Now()
	resp, err := clientUtils.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	rep, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG] 🡱  %dμs POST %s\n  🡱  %s\n  🡳  %s\n", us, nextURL, payload, rep)

	if resp.StatusCode != 200 {
		err = newStatusError(200, resp.Status)
		log.Println("[ERR]", err)
	}
	return
}
