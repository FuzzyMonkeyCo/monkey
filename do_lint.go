package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/googleapis/gnostic/OpenAPIv3"
	"github.com/googleapis/gnostic/compiler"
	"gopkg.in/yaml.v2"
)

func doLint(cfg *ymlCfg, showSpec bool) (spec *sut, err error) {
	docPath, err := cfg.findBlobs()
	if err != nil {
		return
	}

	log.Println("[NFO] reading spec from", docPath)
	docBytes, err := ioutil.ReadFile(docPath)
	if err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Could not read '%s'\n", docPath)
		return
	}

	log.Printf("[NFO] reading info in %dB", len(docBytes))
	info, err := compiler.ReadInfoFromBytes(docPath, docBytes)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Println("[NFO] unpacking info")
	infoMap, ok := compiler.UnpackMap(info)
	if !ok {
		err = errors.New("format:unknown")
		log.Println("[ERR]", err)
		return
	}
	log.Println("[NFO] verifying format is supported", cfg.Kind)
	openapi, ok := compiler.MapValueForKey(infoMap, "openapi").(string)
	if !ok || !strings.HasPrefix(openapi, "3.0") {
		err = errors.New("format:unsupported")
		log.Println("[ERR]", err)
		colorERR.Printf("Format of '%s' is not supported", docPath)
		return
	}

	log.Println("[NFO] parsing whole spec")
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

	log.Println("[NFO] ensuring references are valid")
	if _, err = doc.ResolveReferences(docPath); err != nil {
		log.Println("[ERR]", err)
		colorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR ")
			fmt.Printf("%d: %s\n", 1+i, colorERR.Sprintf(e))
		}
		colorWRN.Println("Documentation validation failed.")
	}

	log.Println("[NFO] preparing spec")
	rawInfo, ok := doc.ToRawInfo().(yaml.MapSlice)
	if !ok {
		rawInfo = nil
	}
	if rawInfo == nil {
		err = errors.New("empty gnostic doc")
		log.Println("[ERR]", err)
		return
	}
	spec = newSUTFromOpenAPIv3(rawInfo)

	// log.Println("[NFO] serialyzing spec to YAML")
	// bytes, err := yaml.Marshal(rawInfo)
	// fmt.Printf("%s %#v\n", bytes, err)

	return
}

type sut struct {
}

func newSUTFromOpenAPIv3(spec yaml.MapSlice) *sut {
	log.Printf(">>> %#v\n", spec)

	return &sut{}
}
