package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	// "archive/tar" FIXME: tar + gz then upload read only conf

	"gopkg.in/yaml.v2"
)

const (
	coveredci = ".coveredci"
	localYML  = coveredci + ".yml"
	mimeJSON  = "application/json"
	mimeYAML  = "application/x-yaml"
)

type ymlCfg struct {
	LaneId uint64
	Script map[string][]string
}

func initDialogue() (*ymlCfg, aCmd) {
	yml := readYAML(localYML)

	// Has to be a string cause []byte gets base64-encoded
	fixtures := map[string]string{localYML: string(yml)}
	payload, err := json.Marshal(fixtures)
	if err != nil {
		log.Fatal(err)
	}
	cmdJSON, laneId := initPUT(payload)
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
		LaneId: laneId,
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

func initPUT(JSON []byte) ([]byte, uint64) {
	var r *http.Request
	r, err := http.NewRequest(http.MethodPut, initURL, bytes.NewBuffer(JSON))
	if err != nil {
		log.Fatal("!initPUT: ", err)
	}

	r.Header.Set("Content-Type", mimeYAML)
	r.Header.Set("Accept", mimeJSON)
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

	laneId, err := strconv.ParseUint(resp.Header.Get("X-Lane-Id"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	return body, laneId
}

func nextPOST(cfg *ymlCfg, payload []byte) []byte {
	if cfg.LaneId == 0 {
		log.Fatal("LaneId is unset")
	}
	URL := nextURL + "/" + fmt.Sprintf("%d", cfg.LaneId)

	r, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal("!nextPOST: ", err)
	}

	r.Header.Set("content-type", mimeJSON)
	r.Header.Set("Accept", mimeJSON)
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
	log.Printf("🡱  %vμs POST %s\n  🡱  %s\n  🡳  %s\n", us, URL, payload, body)

	if resp.StatusCode != 200 {
		log.Fatal("!200: ", resp.Status)
	}

	return body
}
