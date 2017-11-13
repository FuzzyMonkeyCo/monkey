package main

import (
	"encoding/json"
	"log"

	"gopkg.in/xeipuuv/gojsonschema.v0"
)

type aCmd interface {
	Kind() string
	Exec(cfg *ymlCfg) []byte
}

func unmarshalCmd(cmdJSON []byte) aCmd {
	if isValid(schemaREQv1, cmdJSON) {
		var cmd reqCmd
		if err := json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Fatal("[ERR] ", err)
		}
		return cmd
	}

	if isValid(schemaCMDv1, cmdJSON) {
		var cmd simpleCmd
		if err := json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Fatal("[ERR] ", err)
		}
		return cmd
	}

	if isValid(schemaCMDDonev1, cmdJSON) {
		var cmd doneCmd
		if err := json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Fatal("[ERR] ", err)
		}
		return cmd
	}

	return nil // unreachable
}

func isValid(schema string, cmdData []byte) bool {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewStringLoader(string(cmdData))
	validation, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Fatal("[ERR] ", err)
		return false
	}
	return validation.Valid()
}
