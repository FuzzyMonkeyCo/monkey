package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/go-yaml/yaml"

	"github.com/googleapis/gnostic/OpenAPIv3"
	"github.com/googleapis/gnostic/compiler"
	"strings"
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
		fmt.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR $root.")
			fmt.Printf("%d: %s\n", 1+i, e)
		}
		fmt.Println("Documentation validation failed.")
		return
	}

	rawInfo, ok := doc.ToRawInfo().(yaml.MapSlice)
	if !ok || rawInfo == nil {
		err = fmt.Errorf("!yaml")
		log.Println("[ERR]", err)
		return
	}
	if bytes, err = yaml.Marshal(rawInfo); err != nil {
		log.Println("[ERR]", err)
		return
	}
	fmt.Println(bytes)

	_, err = doc.ResolveReferences(docPath)

	rawInfo, ok = doc.ToRawInfo().(yaml.MapSlice)
	if !ok || rawInfo == nil {
		err = fmt.Errorf("!yaml")
		log.Println("[ERR]", err)
		return
	}
	if bytes, err = yaml.Marshal(rawInfo); err != nil {
		log.Println("[ERR]", err)
		return
	}
	fmt.Println(bytes)

	return
}
