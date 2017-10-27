package main

import (
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
	"log"
)

type aCmd interface {
	Kind() string
	Exec(cfg *ymlCfg) []byte
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
