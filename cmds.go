package main

import (
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
	"log"
)

///go:generate stringer -type=base

type cmd byte

const (
	CmdStart1 cmd = iota
	CmdReset1
	CmdStop1
	CmdReq1
)

type CmdRep1 struct {
	Cmd   string  `json:"cmd"`
	V     uint    `json:"v"`
	Us    uint64  `json:"us"`
	Error *string `json:"error"`
}

type Req1 struct {
	UID     interface{} `json:"uid"`
	Method  string      `json:"method"`
	Url     string      `json:"url"`
	Headers []string    `json:"headers"`
	Payload *string     `json:"payload"`
}

type repOK1 struct {
	Cmd     string      `json:"cmd"`
	V       uint        `json:"v"`
	Us      uint64      `json:"us"`
	UID     interface{} `json:"uid"`
	Code    int         `json:"code"`
	Headers []string    `json:"headers"`
	Payload string      `json:"payload"`
}

type repKO1 struct {
	Cmd    string      `json:"cmd"`
	V      uint        `json:"v"`
	Us     uint64      `json:"us"`
	UID    interface{} `json:"uid"`
	Reason string      `json:"reason"`
}

func (cmd cmd) toString() string {
	switch cmd {
	case CmdStart1:
		return "start"
	case CmdReset1:
		return "reset"
	case CmdStop1:
		return "stop"
	case CmdReq1:
		return "req"
	}
	return "unreachable"
}

func pickCmd(cmdData []byte) cmd {
	if isValid(CMDv1, cmdData) {
		var idCmd struct {
			Name string `json:"cmd"`
		}
		if err := json.Unmarshal(cmdData, &idCmd); err != nil {
			log.Fatal("!decode req: ", err)
		}

		switch idCmd.Name {
		case "start":
			return CmdStart1
		case "reset":
			return CmdReset1
		case "stop":
			return CmdStop1
		}
	}

	if isValid(REQv1, cmdData) {
		return CmdReq1
	} else {
		log.Fatal("!pickCmd from ", cmdData)
		return 42 //unreachable
	}
}

func isValid(schema string, cmdData []byte) bool {
	//FIXME: find a loader that works on []byte or even a buffer

	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewStringLoader(string(cmdData))
	validation, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Fatal("!Validate: ", err)
		return false
	}

	if !validation.Valid() {
		// for _, desc := range validation.Errors() {
		// 	log.Printf("\t- %s\n", desc)
		// }
		return false
	}

	return true
}
