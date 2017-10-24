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
	"strings"
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
	log.Printf("%s %vμs PUT %s\n\t%s\n", Up, us, URLInit, JSON)
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

	return body, laneId
}

func initDialogue() (ymlConfig, []byte) {
	yml := readYAML(YML)
	cfg := ymlConf(yml)
	log.Printf("cfg: %+v\n", cfg)

	fixtures := map[string]string{YML: string(yml)}
	payload, err := json.Marshal(fixtures)
	if err != nil {
		log.Fatal(err)
	}

	cmdData, laneId := initPUT(payload)
	cfg.LaneId = laneId
	return cfg, cmdData
}

func next(cfg ymlConfig, cmdData []byte) ([]byte, bool) {
	var rep []byte
	var err error
	switch cmd := pickCmd(cmdData); cmd {
	case CmdReq1:
		ok, ko := makeRequest(cmdData)
		if ok != nil {
			rep, err = json.Marshal(*ok)
		} else {
			rep, err = json.Marshal(*ko)
		}
	case CmdStart1, CmdReset1, CmdStop1:
		cmdRet := executeScript(cmd, cfg)
		rep, err = json.Marshal(cmdRet)
	}

	if err != nil {
		log.Fatal("!encode: ", err)
	}

	return nextPOST(cfg, rep), false
}

func makeRequest(cmdData []byte) (*repOK1, *repKO1) {
	var req Req1
	if err := json.Unmarshal(cmdData, &req); err != nil {
		log.Fatal("!decode req1: ", err)
	}

	var r *http.Request
	var err error
	if req.Payload != nil {
		inPayload := bytes.NewBufferString(*req.Payload)
		r, err = http.NewRequest(req.Method, req.Url, inPayload)
	} else {
		r, err = http.NewRequest(req.Method, req.Url, nil)
	}
	if err != nil {
		log.Fatal("!NewRequest: ", err)
	}
	for _, header := range req.Headers {
		if header == "User-Agent: CoveredCI-passthrough/1" {
			r.Header.Set("User-Agent", Version)
		} else {
			pair := strings.SplitN(header, ": ", 2)
			r.Header.Set(pair[0], pair[1])
		}
	}
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		reason := fmt.Sprintf("%+v", err.Error())
		log.Printf("%s %vμs %s %s\n\t%s\n", Down, us, req.Method, req.Url, reason)
		rep := &repKO1{
			UID:    req.UID,
			V:      1,
			Us:     us,
			Reason: reason,
		}
		return nil, rep
	} else {

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("!read body: ", err)
		}
		log.Printf("%s %vμs %s %s\n\t%s\n", Down, us, req.Method, req.Url, body)
		var headers []string
		//// headers = append(headers, fmt.Sprintf("Host: %v", resp.Host))
		// Loop through headers
		//FIXME: preserve order github.com/golang/go/issues/21853
		for name, values := range resp.Header {
			name = strings.ToLower(name)
			for _, value := range values {
				headers = append(headers, fmt.Sprintf("%v: %v", name, value))
			}
		}

		rep := &repOK1{
			UID:     req.UID,
			V:       1,
			Us:      us,
			Code:    resp.StatusCode,
			Headers: headers,
			Payload: string(body),
		}
		return rep, nil
	}
}

func executeScript(cmd cmd, cfg ymlConfig) CmdRep1 {
	log.Println(cfg) //FIXME
	// if len(cfg.) == 0 {
	return CmdRep1{
		Cmd:   cmd.toString(),
		V:     1,
		Us:    0,
		Error: nil,
	}
	// }
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
	log.Printf("%s %vμs POST %s\n\t%s\n", Up, us, URL, payload)
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

	return body
}
