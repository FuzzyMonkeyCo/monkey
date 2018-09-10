package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"

	"github.com/xeipuuv/gojsonschema"
)

var errInvalidPayload = errors.New("invalid JSON payload")
var errNoSuchRef = errors.New("no such $ref")

type sid = uint32
type schemaJSON = map[string]interface{}
type schemasJSON = map[string]schemaJSON

type validator struct {
	Spec *SpecIR
	Refs map[string]sid
	Refd *gojsonschema.SchemaLoader
	Anon map[sid]schemaJSON
}

func newValidator(capaEndpoints, capaSchemas int) *validator {
	return &validator{
		Refs: make(map[string]sid, capaSchemas),
		Anon: make(map[sid]schemaJSON, capaEndpoints),
		Spec: &SpecIR{
			Endpoints: make([]*Endpoint, 0, capaEndpoints),
			Schemas:   &Schemas{Json: make(map[sid]*RefOrSchemaJSON, capaSchemas)},
		},
	}
}

func (vald *validator) newSID() sid {
	return sid(1 + len(vald.Spec.Schemas.Json))
}

func (vald *validator) seed(base string, schemas schemasJSON) (err error) {
	i, names := 0, make([]string, len(schemas))
	for name := range schemas {
		names[i] = name
		i++
	}
	sort.Strings(names)

	for j := 0; j != i; j++ {
		name := names[j]
		schema, absRef := schemas[name], base+name
		log.Printf("[DBG] seeding schema '%s'", absRef)

		sl := gojsonschema.NewGoLoader(schema)
		if err = vald.Refd.AddSchema(absRef, sl); err != nil {
			log.Println("[ERR]", err)
			return
		}

		sid := vald.ensureMapped("", schema)
		vald.Refs[absRef] = sid
		schemaPtr := &SchemaPtr{Ref: absRef, SID: sid}
		vald.Spec.Schemas.Json[vald.newSID()] = &RefOrSchemaJSON{
			PtrOrSchema: &RefOrSchemaJSON_Ptr{schemaPtr},
		}
	}
	return
}

func (vald *validator) ensureMapped(ref string, goSchema schemaJSON) sid {
	if ref == "" {
		schema := vald.fromGo(goSchema)
		for SID, schemaPtr := range vald.Spec.Schemas.Json {
			//TODO: drop reflect
			if s := schemaPtr.GetSchema(); s != nil && reflect.DeepEqual(schema, s) {
				return SID
			}
		}
		SID := vald.newSID()
		vald.Spec.Schemas.Json[SID] = &RefOrSchemaJSON{
			PtrOrSchema: &RefOrSchemaJSON_Schema{&schema},
		}
		return SID
	}

	mappedSID, ok := vald.Refs[ref]
	if !ok {
		panic(`no such ref:` + ref)
	}
	schemaPtr := &SchemaPtr{Ref: ref, SID: mappedSID}
	SID := sm.newSID()
	vald.Spec.Schemas.Json[SID] = &RefOrSchemaJSON{
		PtrOrSchema: &RefOrSchemaJSON_Ptr{schemaPtr},
	}
	return SID
}

func (vald *validator) validationErrors(spec *SpecIR, SID sid) []error {
	//FIXME: build schemaJSON from SID & SIDs then compile against schemaLoader
}

func (vald *validator) validateAgainstSchema(absRef string) (err error) {
	if _, ok := schemas[absRef]; !ok {
		err = errNoSuchRef
		return
	}

	var value interface{}
	if err = json.NewDecoder(os.Stdin).Decode(&value); err != nil {
		log.Println("[ERR]", err)
		return
	}

	// NOTE Compile errs on bad refs only, MUST do this step in `lint`
	schema, err := vald.Refd.Compile(
		gojsonschema.NewGoLoader(schemaJSON{"$ref": absRef}))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	res, err := schema.Validate(gojsonschema.NewGoLoader(value))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	errs := res.Errors()
	for _, e := range errs {
		// ResultError interface
		colorERR.Println(e)
	}
	if len(errs) > 0 {
		err = errInvalidPayload
	}
	return
}
