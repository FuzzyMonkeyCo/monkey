package main

import (
	"errors"
	"log"

	o "github.com/googleapis/gnostic/OpenAPIv3"
)

func newSpecFromOpenAPIv3(doc *o.Document) (spec *SpecIR, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	basePath, err := specBasePath(doc)
	if err != nil {
		return
	}
	log.Println(basePath)

	spec = &SpecIR{}
	log.Println(spec.String())
	return
}

//TODO: support the whole spec on /"servers"
func specBasePath(doc *o.Document) (basePath string, err error) {
	if len(doc.Servers) == 0 {
		log.Println(`[NFO] field 'servers' empty/unset: using "/"`)
		basePath = "/"
		return
	}

	if len(doc.Servers) != 1 {
		log.Println(`[NFO] field 'servers' has many values: using the first one`)
	}
	basePath = doc.Servers[0].Url
	if basePath == "" || basePath[0] != '/' {
		err = errors.New(`field 'servers' has no suitable 'url'`)
		log.Println("[ERR]", err)
		colorERR.Println(err)
	}
	return
}
