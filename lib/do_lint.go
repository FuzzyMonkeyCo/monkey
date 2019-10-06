package lib

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gogo/protobuf/jsonpb"
	openapi_v3 "github.com/googleapis/gnostic/OpenAPIv3"
	"github.com/googleapis/gnostic/compiler"
	"gopkg.in/yaml.v2"
)

// DoLint builds a valid spec IR
func DoLint(docPath string, blob []byte, showSpec bool) (vald *Validator, err error) {
	log.Printf("[NFO] reading info in %dB", len(blob))
	if err = validateAndPretty(docPath, blob, showSpec); err != nil {
		return
	}

	loader := openapi3.NewSwaggerLoader()
	doc, err := loader.LoadSwaggerFromData(blob)
	if err != nil {
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}

	log.Println("[NFO] first validation pass")
	if err = doc.Validate(loader.Context); err != nil {
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}

	log.Println("[NFO] last validation pass")
	if vald, err = newSpecFromOA3(doc); err != nil {
		return
	}

	log.Println("[DBG] serializing the protobuf")
	jsoner := &jsonpb.Marshaler{
		// Indent: "\t",
		// EmitDefaults: true,
	}
	stringified, err := jsoner.MarshalToString(vald.Spec)
	log.Printf("[DBG] %+v %s...", err, stringified[:37])
	return
}

func validateAndPretty(docPath string, blob []byte, showSpec bool) (err error) {
	info, err := compiler.ReadInfoFromBytes(docPath, blob)
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
	log.Println("[NFO] verifying format is supported")
	openapi, ok := compiler.MapValueForKey(infoMap, "openapi").(string)
	if !ok || !strings.HasPrefix(openapi, "3.0") {
		err = errors.New("format:unsupported")
		log.Println("[ERR]", err)
		ColorERR.Printf("Format of '%s' is not supported", docPath)
		return
	}

	log.Println("[NFO] parsing whole spec")
	doc, err := openapi_v3.NewDocument(info, compiler.NewContext("$root", nil))
	if err != nil {
		log.Println("[ERR]", err)
		ColorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR $root.")
			fmt.Printf("%d: %s\n", 1+i, ColorERR.Sprintf(e))
		}
		ColorERR.Println("Documentation validation failed.")
		return
	}

	log.Println("[NFO] ensuring references are valid")
	if _, err = doc.ResolveReferences(docPath); err != nil {
		log.Println("[ERR]", err)
		ColorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR ")
			fmt.Printf("%d: %s\n", 1+i, ColorERR.Sprintf(e))
		}
		ColorERR.Println("Documentation validation failed.")
		return
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

	if showSpec {
		log.Println("[NFO] serialyzing spec to YAML")
		ColorNFO.Println("Spec:")
		var pretty []byte
		if pretty, err = yaml.Marshal(rawInfo); err != nil {
			log.Println("[ERR]", err)
			return
		}
		fmt.Fprintf(os.Stderr, "%s\n", pretty)
	}
	return
}
