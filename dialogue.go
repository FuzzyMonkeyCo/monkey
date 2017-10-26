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
	//FIXME use HTTPS
	UPSTREAM = "http://localhost:1042"
	URLInit  = UPSTREAM + "/1/init"
	URLNext  = UPSTREAM + "/1/next"
	YML      = ".coveredci.yml"
	Up       = "🡱"
	Down     = "🡳"
)

type ymlConfig struct {
	LaneId uint64
	Start  []string `yaml:"start"`
	Reset  []string `yaml:"reset"`
	Stop   []string `yaml:"stop"`
}

func initDialogue() (ymlConfig, aCmd) {
	yml := readYAML(YML)
	cfg := ymlConf(yml)
	log.Printf("cfg: %+v\n", cfg)

	fixtures := map[string]string{YML: string(yml)}
	payload, err := json.Marshal(fixtures)
	if err != nil {
		log.Fatal(err)
	}

	cmdJSON, laneId := initPUT(payload)
	cmd := unmarshalCmd(cmdJSON)
	cfg.LaneId = laneId
	return cfg, cmd
}

func next(cfg ymlConfig, cmd aCmd) aCmd {
	rep := cmd.Exec(cfg)
	if cmd.Kind() == "done" {
		return nil
	}
	nextCmdJSON := nextPOST(cfg, rep)
	return unmarshalCmd(nextCmdJSON)
}

func readYAML(path string) []byte {
	fd, err := os.Open(path)
	if err != nil {
		log.Fatal("!fd: ", err)
	}
	defer fd.Close()

	yml, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Fatal("!yml: ", err)
	}

	return yml
}

func ymlConf(yml []byte) ymlConfig {
	var cfg ymlConfig
	err := yaml.Unmarshal(yml, &cfg)
	if err != nil {
		log.Fatal("!cfg: ", err)
	}
	return cfg
}

func initPUT(JSON []byte) ([]byte, uint64) {
	var r *http.Request
	r, err := http.NewRequest(http.MethodPut, URLInit, bytes.NewBuffer(JSON))
	if err != nil {
		log.Fatal("!initPUT: ", err)
	}

	r.Header.Set("Content-Type", "application/x-yaml")
	r.Header.Set("Accept", "application/json")
	client := &http.Client{}

	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Fatal("!PUT: ", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		log.Fatal("!201: ", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("!read body: ", err)
	}

	laneId, err := strconv.ParseUint(resp.Header.Get("X-Lane-Id"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s %vμs PUT %s\n\t%s\n%v\n\t\t%s\n", Up, us, URLInit, JSON, laneId, body)

	return body, laneId
}

func nextPOST(cfg ymlConfig, payload []byte) []byte {
	if cfg.LaneId == 0 {
		log.Fatal("LaneId is unset")
	}
	URL := URLNext + "/" + fmt.Sprintf("%d", cfg.LaneId)

	r, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal("!nextPOST: ", err)
	}

	r.Header.Set("content-type", "application/json")
	r.Header.Set("Accept", "application/json")
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		log.Fatal("!POST: ", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal("!200: ", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("!read body: ", err)
	}
	log.Printf("%s %vμs POST %s\n\t%s\n\t\t%s\n", Up, us, URL, payload, body)

	return body
}
