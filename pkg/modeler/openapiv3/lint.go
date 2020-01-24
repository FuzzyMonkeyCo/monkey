package modeler_openapiv3

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/getkin/kin-openapi/openapi3"
	openapi_v3 "github.com/googleapis/gnostic/OpenAPIv3"
	"github.com/googleapis/gnostic/compiler"
	"gopkg.in/yaml.v2"
)

// Lint TODO
func (m *oa3) Lint(ctx context.Context, showSpec bool) (err error) {
	var blob []byte
	if blob, err = ioutil.ReadFile(m.File); err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[NFO] reading info in %dB", len(blob))
	if err = validateAndPretty(m.File, blob, showSpec); err != nil {
		return
	}

	loader := &openapi3.SwaggerLoader{
		Context:                ctx,
		IsExternalRefsAllowed:  true,
		LoadSwaggerFromURIFunc: m.loadSwaggerFromURI,
	}
	doc, err := loader.LoadSwaggerFromData(blob)
	if err != nil {
		log.Println("[ERR]", err)
		as.ColorERR.Println(err)
		return
	}

	log.Println("[NFO] first validation pass")
	if err = doc.Validate(ctx); err != nil {
		log.Println("[ERR]", err)
		as.ColorERR.Println(err)
		return
	}

	log.Println("[NFO] last validation pass")
	if m.vald, err = newSpecFromOA3(doc); err != nil {
		return
	}

	log.Println("[NFO] model is valid")
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
		as.ColorERR.Printf("Format of '%s' is not supported", docPath)
		return
	}

	log.Println("[NFO] parsing whole spec")
	doc, err := openapi_v3.NewDocument(info, compiler.NewContext("$root", nil))
	if err != nil {
		log.Println("[ERR]", err)
		as.ColorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR $root.")
			fmt.Printf("%d: %s\n", 1+i, as.ColorERR.Sprintf(e))
		}
		as.ColorERR.Println("Documentation validation failed.")
		return
	}

	log.Println("[NFO] ensuring references are valid")
	if _, err = doc.ResolveReferences(docPath); err != nil {
		log.Println("[ERR]", err)
		as.ColorWRN.Println("Validation errors:")
		for i, line := range strings.Split(err.Error(), "\n") {
			e := strings.TrimPrefix(line, "ERROR ")
			fmt.Printf("%d: %s\n", 1+i, as.ColorERR.Sprintf(e))
		}
		as.ColorERR.Println("Documentation validation failed.")
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
		as.ColorNFO.Println("Spec:")
		var pretty []byte
		if pretty, err = yaml.Marshal(rawInfo); err != nil {
			log.Println("[ERR]", err)
			return
		}
		fmt.Fprintf(os.Stderr, "%s\n", pretty)
	}
	return
}

func (m *oa3) loadSwaggerFromURI(loader *openapi3.SwaggerLoader, uri *url.URL) (*openapi3.Swagger, error) {
	// TODO: support local & remote URIs
	return nil, fmt.Errorf("unsupported URI: %q", uri.String())
}
