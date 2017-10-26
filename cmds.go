package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type aCmd interface {
	Kind() string
	Exec(cfg ymlConfig) []byte
}

type simpleCmd struct {
	V   uint   `json:"v"`
	Cmd string `json:"cmd"`
}

type reqCmd struct {
	V       uint        `json:"v"`
	Cmd     string      `json:"cmd"`
	UID     interface{} `json:"uid"`
	Method  string      `json:"method"`
	Url     string      `json:"url"`
	Headers []string    `json:"headers"`
	Payload *string     `json:"payload"`
}

type simpleCmdRep struct {
	Cmd   string  `json:"cmd"`
	V     uint    `json:"v"`
	Us    uint64  `json:"us"`
	Error *string `json:"error"`
}

type reqCmdRepOK struct {
	Cmd     string      `json:"cmd"`
	V       uint        `json:"v"`
	Us      uint64      `json:"us"`
	UID     interface{} `json:"uid"`
	Code    int         `json:"code"`
	Headers []string    `json:"headers"`
	Payload string      `json:"payload"`
}

type reqCmdRepKO struct {
	Cmd    string      `json:"cmd"`
	V      uint        `json:"v"`
	Us     uint64      `json:"us"`
	UID    interface{} `json:"uid"`
	Reason string      `json:"reason"`
}

func (cmd simpleCmd) Kind() string {
	return cmd.Cmd
}

func (cmd simpleCmd) Exec(cfg ymlConfig) []byte {
	cmdRet := executeScript(cfg, cmd)
	rep, err := json.Marshal(cmdRet)
	if err != nil {
		log.Fatal(err)
	}
	return rep
}

func executeScript(cfg ymlConfig, cmd simpleCmd) *simpleCmdRep {
	kind := cmd.Kind()
	cmds := scriptSetting(cfg, kind)
	if len(cmds) == 0 {
		return &simpleCmdRep{V: 1, Cmd: kind}
	}

	script := strings.Join(cmds, " && ") //FIXME: escape newlines and that's it?
	shell := os.Getenv("SHELL")
	var stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	log.Println("Executing:", script)
	exe := exec.CommandContext(ctx, shell, "-e", "-o", "pipefail", "-c", script)
	exe.Env = append(os.Environ(), "COVEREDCI=true")
	exe.Stdout = os.Stdout
	exe.Stderr = &stderr

	err := exe.Run()
	if err != nil {
		error := string(stderr.Bytes()) + err.Error()
		return &simpleCmdRep{V: 1, Cmd: kind, Error: &error}
	} else {
		return &simpleCmdRep{V: 1, Cmd: kind}
	}
}

func scriptSetting(cfg ymlConfig, cmdKind string) []string {
	switch cmdKind {
	case "reset":
		return cfg.Reset
	case "start":
		return cfg.Start
	}
	return cfg.Stop
}

func (cmd reqCmd) Kind() string {
	return cmd.Cmd
}

func (cmd reqCmd) Exec(_cfg ymlConfig) []byte {
	ok, ko := makeRequest(cmd)

	var rep []byte
	var err error
	if ok != nil {
		rep, err = json.Marshal(ok)
	} else {
		rep, err = json.Marshal(ko)
	}

	if err != nil {
		log.Fatal("!encode: ", err)
	}
	return rep
}

func makeRequest(cmd reqCmd) (*reqCmdRepOK, *reqCmdRepKO) {
	var r *http.Request
	var err error
	if cmd.Payload != nil {
		inPayload := bytes.NewBufferString(*cmd.Payload)
		r, err = http.NewRequest(cmd.Method, cmd.Url, inPayload)
	} else {
		r, err = http.NewRequest(cmd.Method, cmd.Url, nil)
	}
	if err != nil {
		log.Fatal("!NewRequest: ", err)
	}
	for _, header := range cmd.Headers {
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
		log.Printf("%s %vμs %s %s\n\t%s\n", Down, us, cmd.Method, cmd.Url, reason)
		ko := &reqCmdRepKO{
			V:      1,
			Cmd:    cmd.Cmd,
			UID:    cmd.UID,
			Us:     us,
			Reason: reason,
		}
		return nil, ko

	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("!read body: ", err)
		}
		log.Printf("%s %vμs %s %s\n\t%s\n", Down, us, cmd.Method, cmd.Url, body)
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

		ok := &reqCmdRepOK{
			V:       1,
			Cmd:     cmd.Cmd,
			UID:     cmd.UID,
			Us:      us,
			Code:    resp.StatusCode,
			Headers: headers,
			Payload: string(body),
		}
		return ok, nil
	}
}

func unmarshalCmd(cmdJSON []byte) aCmd {
	if isValid(CMDv1, cmdJSON) {
		var cmd simpleCmd
		if err := json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Fatal(err)
		}
		return cmd
	}

	if isValid(REQv1, cmdJSON) {
		var cmd reqCmd
		if err := json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Fatal(err)
		}
		return cmd
	}

	return nil //unreachable
}

func isValid(schema string, cmdData []byte) bool {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewStringLoader(string(cmdData))
	validation, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Fatal(err)
		return false
	}
	return validation.Valid()
}
