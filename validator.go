package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

var errInvalidPayload = errors.New("invalid JSON payload")
var errNoSuchRef = errors.New("no such $ref")

type sid = uint32
type schemaJSON = map[string]interface{}
type schemasJSON = map[string]schemaJSON
type refOrSchemaJSON struct {
	SID    sid
	Ref    string
	Schema *schemaJSON
}

type validator struct {
	// SIDs map[sid]refOrSchemaJSON
	Schemas schemasJSON
}

// func newValidator(capa int) *validator {
// 	return &validator{
// 		SIDs: make(map[sid]refOrSchemaJSON, capa),
// 	}
// }

// func (vald *validator) newSID() sid {
// 	return sid(1 + len(vald.SIDs))
// }

// func (vald *validator) seed(ref string, schema schemaJSON) {
// 	RoS := vald.ensureMapped("", schema)
// 	RoS.Ref = ref
// 	vald.SIDs[vald.newSID()] = RoS
// }

// func (vald *validator) ensureMapped(ref string, schema schemaJSON) sid {
// 	if ref == "" {
// 		for SID, RoS := range vald.SIDs {
// 			if s := RoS.Schema(); s != nil && reflect.DeepEqual(schema, s) {
// 				return SID
// 			}
// 		}
// 		SID := vald.newSID()
// 		vald.SIDs[SID] = &refOrSchemaJSON{Schema: schema}
// 		return SID
// 	}

// 	if schema == nil {
// 		panic("no ref nor schema!")
// 	}

// 	mappedSID := sid(0)
// 	for SID, RoS := range vald.SIDs {
// 		if RoS.ref == ref {
// 			mappedSID = SID
// 			break
// 		}
// 	}
// 	RoS := &refOrSchemaJSON{Ref: ref, SID: mappedSID}
// 	SID := vald.newSID()
// 	vald.SIDs[SID] = RoS
// 	return SID
// }

func useSpecSchemas(spec *SpecIR, vald *validator) (err error) {
	//FIXME: put vald into spec
	return
}

func printSchemaRefs(schemas schemasJSON) {
	for absRef := range schemas {
		fmt.Println(absRef)
	}
}

func validateAgainstSchema(schemas schemasJSON, absRef string) (err error) {
	if _, ok := schemas[absRef]; !ok {
		err = errNoSuchRef
		return
	}

	var value interface{}
	if err = json.NewDecoder(os.Stdin).Decode(&value); err != nil {
		log.Println("[ERR]", err)
		return
	}

	sl := gojsonschema.NewSchemaLoader()
	for schemaRef, schema := range schemas {
		l := gojsonschema.NewGoLoader(schema)
		if err = sl.AddSchema(schemaRef, l); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}

	// NOTE Compile errs on bad refs only, MUST do this step in `lint`
	schema, err := sl.Compile(
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
