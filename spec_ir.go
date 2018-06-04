package main

import (
	"log"

	"gopkg.in/yaml.v2"
)

type specIR struct {
}

func newSpecFromOpenAPIv3(openAPI yaml.MapSlice) (spec *specIR, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	spec = &specIR{}
	return
}
