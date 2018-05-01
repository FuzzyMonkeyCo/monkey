package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/googleapis/gnostic/OpenAPIv3"
	"github.com/googleapis/gnostic/compiler"
)

func lintDocs(cfg *ymlCfg, apiKey string) (bytes []byte, err error) {
	docPath, err := cfg.findBlobs()
	if err != nil {
		return
	}

	docBytes, err := ioutil.ReadFile(docPath)
	if err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Could not read '%s'\n", docPath)
		return
	}

	info, err := compiler.ReadInfoFromBytes(docPath, docBytes)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	infoMap, ok := compiler.UnpackMap(info)
	if !ok {
		err = fmt.Errorf("format:unknown")
		log.Println("[ERR]", err)
		return
	}
	openapi, ok := compiler.MapValueForKey(infoMap, "openapi").(string)
	if !ok || !strings.HasPrefix(openapi, "3.0") {
		err = fmt.Errorf("format:unsupported")
		log.Println("[ERR]", err)
		return
	}

	doc, err := openapi_v3.NewDocument(info, compiler.NewContext("$root", nil))
	if err != nil {
		log.Println("[ERR]", err)
		colorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR $root.")
			fmt.Printf("%d: %s\n", 1+i, colorERR.Sprintf(e))
		}
		colorWRN.Println("Documentation validation failed.")
		return
	}

	if err = dumpGnostic("_", doc); err != nil {
		return
	}
	if _, err = doc.ResolveReferences(docPath); err != nil {
		log.Println("[ERR]", err)
		colorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR ")
			fmt.Printf("%d: %s\n", 1+i, colorERR.Sprintf(e))
		}
		colorWRN.Println("Documentation validation failed.")
		return
	}
	err = dumpGnostic("__", doc)

	return
}

func dumpGnostic(path string, doc *openapi_v3.Document) (err error) {
	raw := doc.ToRawInfo()
	if raw == nil {
		err = fmt.Errorf("!yaml! %+v", raw)
		log.Println("[ERR]", err)
		return
	}

	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer fd.Close()

	if err = yaml.NewEncoder(fd).Encode(raw); err != nil {
		log.Println("[ERR]", err)
		return
	}
	colorNFO.Println(path)
	return
}
