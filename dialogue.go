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

func initDialogue(apiKey string) (*ymlCfg, aCmd, error) {
	yml, err := readYAML(localYML)
	if err != nil {
		return nil, nil, err
	}

	validationJSON, err := validateDocs(apiKey, yml)
	if err != nil {
		return nil, nil, err
	}

	cmdJSON, authToken, err := initPUT(apiKey, validationJSON)
	if err != nil {
		return nil, nil, err
	}
	cmd, err := unmarshalCmd(cmdJSON)
	if err != nil {
		return nil, nil, err
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
	if err := yaml.Unmarshal(yml, &ymlConf); err != nil {
		log.Println("[ERR]", err)
		return nil, nil, err
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
	return cfg, cmd, nil
}

func next(cfg *ymlCfg, cmd aCmd) (aCmd, error) {
	// Sometimes sets cfg.Final* fields
	rep := cmd.Exec(cfg)

	nextCmdJSON, err := nextPOST(cfg, rep)
	if err != nil {
		return nil, err
	}
	return unmarshalCmd(nextCmdJSON)
}

func readYAML(path string) ([]byte, error) {
	fd, err := os.Open(path)
	if err != nil {
		fmt.Println("You must provide a readable '.coveredci.yml' file in the current directory.")
		log.Println("[ERR]", err)
		return nil, err
	}
	defer fd.Close()

	yml, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	return yml, nil
}

func initPUT(apiKey string, JSON []byte) ([]byte, string, error) {
	r, err := http.NewRequest(http.MethodPut, initURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Println("[ERR]", err)
		return nil, "", err
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
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return nil, "", err
	}
	log.Printf("[DBG] 🡱  %vμs PUT %s\n  🡱  %s\n  🡳  %s\n", us, initURL, JSON, body)

	if resp.StatusCode != 201 {
		err := fmt.Errorf("not 201: %v", resp.Status)
		log.Println("[ERR]", err)
		return nil, "", err
	}

	authToken := resp.Header.Get(xAuthTokenHeader)
	if authToken == "" {
		err := fmt.Errorf("Could not acquire an AuthToken")
		log.Println("[ERR]", err)
		fmt.Println(err)
		return nil, "", err
	}

	return body, authToken, nil
}

func nextPOST(cfg *ymlCfg, payload []byte) ([]byte, error) {
	r, err := http.NewRequest(http.MethodPost, nextURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
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
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	log.Printf("[DBG] 🡱  %vμs POST %s\n  🡱  %s\n  🡳  %s\n", us, nextURL, payload, body)

	if resp.StatusCode != 200 {
		err := fmt.Errorf("not 200: %v", resp.Status)
		log.Println("[ERR]", err)
		return nil, err
	}

	return body, nil
}
