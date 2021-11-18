package openapiv3

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	openapi_v3 "github.com/google/gnostic/openapiv3"
)

var errLinting = func() error {
	msg := "Documentation validation failed."
	return errors.New(msg) // Gets around golint
}()

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

	loader := &openapi3.Loader{
		Context:               ctx,
		IsExternalRefsAllowed: true,
		ReadFromURIFunc:       m.readFromURI,
	}
	doc, err := loader.LoadFromData(blob)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] first validation pass")
	if err = doc.Validate(ctx); err != nil {
		log.Println("[ERR]", err)
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
		for _, line := range strings.Split(err.Error(), "\n") {
			es := strings.SplitAfterN(line, "$root.", 2) // TODO: handle line:col
			fmt.Println(es[1])
		}
		err = errLinting
		return
	}

	log.Println("[NFO] ensuring references are valid")
	if _, err = doc.ResolveReferences(docPath); err != nil {
		log.Println("[ERR]", err)
		for _, line := range strings.Split(err.Error(), "\n") {
			fmt.Println(strings.TrimPrefix(line, "ERROR "))
		}
		err = errLinting
		return
	}

	if showSpec {
		log.Println("[NFO] serialyzing spec to YAML")
		var pretty []byte
		if pretty, err = doc.YAMLValue(""); err != nil {
			log.Println("[ERR]", err)
			return
		}
		fmt.Fprintf(os.Stderr, "%s\n", pretty)
	}
	return
}

func (m *oa3) readFromURI(loader *openapi3.Loader, uri *url.URL) ([]byte, error) {
	// TODO: support local & remote URIs
	return nil, fmt.Errorf("unsupported URI: %q", uri.String())
}
