package main

import (
	"os"
	"bytes"
	"net/http"
	"time"
	"log"
	"io/ioutil"
	// "archive/tar" FIMXE: tar + gz then upload read only conf
	"gopkg.in/yaml.v2"
)

const (
	UPSTREAM = "http://localhost:1042"
	URLInit = UPSTREAM + "/1/init"
	URLNext = UPSTREAM + "/1/next"
	YML = ".coveredci.yml"
)

type ymlConfig struct {
	Start []string `yaml:"start"`
	Reset []string `yaml:"reset"`
	Stop []string `yaml:"stop"`
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

func initPUT(yml []byte) []byte {
	var r *http.Request
	payload := bytes.NewBuffer(yml)
	r, err := http.NewRequest(http.MethodPut, URLInit, payload)
	if err != nil {
		log.Fatal("!initPUT: ", err)
	}

	r.Header.Set("Content-Type", "application/x-yaml")
	r.Header.Set("Accept", "application/json")
	client := &http.Client{}

	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	log.Printf("%vμs PUT %s\n", us, URLInit)
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

	return body
}

func initDialogue() []byte {
	yml := readYAML(YML)
	cfg := ymlConf(yml)
	log.Printf("cfg: %+v\n", cfg)

	cmdData := initPUT(yml)
	return cmdData
}

