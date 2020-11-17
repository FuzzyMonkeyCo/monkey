package openapiv3

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/getkin/kin-openapi/openapi3"
	openapi_v3 "github.com/googleapis/gnostic/openapiv3"
)

// Lint goes through OpenAPIv3 specs and unsures they're valid
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
	log.Println("[NFO] parsing whole spec")
	doc, err := openapi_v3.ParseDocument(blob)
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

	if showSpec {
		log.Println("[NFO] serialyzing spec to YAML")
		as.ColorNFO.Println("Spec:")
		var pretty []byte
		if pretty, err = doc.YAMLValue(""); err != nil {
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
