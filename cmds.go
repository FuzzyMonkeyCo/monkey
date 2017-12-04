package main

import (
	"encoding/json"
	"fmt"
	"log"
)

var (
	lastLane      lane
	shrinkingFrom lane
	totalR        uint
)

type lane struct {
	T uint `json:"t"`
	R uint `json:"r"`
}

type aCmd interface {
	Kind() string
	Exec(cfg *ymlCfg) (rep []byte, err error)
}

func unmarshalCmd(cmdJSON []byte) (cmd aCmd, err error) {
	ok, err := isValidForSchemaREQv1(cmdJSON)
	if err != nil {
		return
	}
	if ok {
		var cmd reqCmd
		if err = json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Println("[ERR]", err)
			return nil, err
		}
		return &cmd, nil
	}

	ok, err = isValidForSchemaCMDv1(cmdJSON)
	if err != nil {
		return
	}
	if ok {
		var cmd simpleCmd
		if err = json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Println("[ERR]", err)
			return nil, err
		}
		return &cmd, nil
	}

	ok, err = isValidForSchemaCMDDonev1(cmdJSON)
	if err != nil {
		return
	}
	if ok {
		var cmd doneCmd
		if err = json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Println("[ERR]", err)
			return nil, err
		}
		return &cmd, nil
	}

	err = fmt.Errorf("invalid JSON data received")
	log.Println("[ERR]", err)
	return
}
