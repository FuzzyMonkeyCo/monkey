package main

import (
	"errors"
	"fmt"
	"log"

	o "github.com/googleapis/gnostic/OpenAPIv3"
)

type mapKeyToPtrOrSchema map[string]*PtrOrSchemaJSONDraft05

func newSpecFromOpenAPIv3(doc *o.Document) (spec *SpecIR, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	schemas, err := specSchemas("#/components/schemas/", doc)
	if err != nil {
		return
	}

	basePath, err := specBasePath(doc)
	if err != nil {
		return
	}

	spec = &SpecIR{
		Schemas: &Schemas{
			Schemas_JSONDraft05: schemas,
		},
	}
	log.Printf("\n basePath:%#v\n spec: %v\n ", basePath, spec)
	return
}

func specSchemas(baseRef string, doc *o.Document) (schemas mapKeyToPtrOrSchema, err error) {
	schemas = make(mapKeyToPtrOrSchema)
	docSchemas := doc.GetComponents().GetSchemas().GetAdditionalProperties()
	for _, namedSchema := range docSchemas {
		ptr := baseRef + namedSchema.GetName()
		docSchema := namedSchema.GetValue().GetSchema()
		if docSchema == nil {
			ref := namedSchema.GetValue().GetReference()
			if ref == nil {
				err = fmt.Errorf("%s is neither ref nor schema", ptr)
				log.Println("[ERR]", err)
				return
			}
			schemas[ptr] = &PtrOrSchemaJSONDraft05{
				PtrOrSchema_JSONDraft05: &PtrOrSchemaJSONDraft05_Ptr{Ptr: ref.GetXRef()},
			}
		}

		colorNFO.Printf("%#v\n", docSchema)
		var ptrOrSchema *PtrOrSchemaJSONDraft05
		if ptrOrSchema, err = specSchemaFromDocSchema(ptr, docSchema); err != nil {
			return
		}
		colorERR.Printf("%#v --> %v\n", ptr, ptrOrSchema)
		schemas[ptr] = ptrOrSchema
	}
	return
}

func specSchemaFromDocSchema(ptr string, s *o.Schema) (*PtrOrSchemaJSONDraft05, error) {
	wasSet := false
	schema := &Schema_JSONDraft05{}

	// enum
	// sEnum := s.GetEnum()
	// if len(sEnum) != 0 {
	// 	schema.Enum =
	// }

	// type, nullable
	sType := s.GetType()
	if len(sType) != 0 {
		t := Schema_JSONDraft05_Type(Schema_JSONDraft05_Type_value[sType])
		if s.GetNullable() {
			schema.Type = []Schema_JSONDraft05_Type{t, Schema_JSONDraft05_null}
		} else {
			schema.Type = []Schema_JSONDraft05_Type{t}
		}
		wasSet = true
	}

	// format
	sFormat := s.GetFormat()
	if len(sFormat) != 0 {
		schema.Format = sFormat
		wasSet = true
	}

	// properties, required
	sProperties := s.GetProperties().GetAdditionalProperties()
	if len(sProperties) != 0 {
		specMaybeAddType(Schema_JSONDraft05_object, schema.Type)
		schema.Required = s.GetRequired()
		schema.Properties = make(mapKeyToPtrOrSchema)
		for _, namedSchema := range sProperties {
			name := namedSchema.GetName()
			subPtr := ptr + "/" + name
			subSchema := namedSchema.GetValue().GetSchema()
			if subSchema == nil {
				err := fmt.Errorf("%s is a ref and that's not yet supported", subPtr)
				log.Println("[ERR]", err)
				log.Printf("[ERR] >>> %#v\n", namedSchema.GetValue().GetReference())
				return nil, err
			}

			subS, err := specSchemaFromDocSchema(subPtr, subSchema)
			if err != nil {
				return nil, err
			}
			schema.Properties[name] = subS
		}
		wasSet = true
	}

	if !wasSet {
		err := fmt.Errorf("%s is an empty schema: %#v", ptr, s)
		log.Println("[ERR]", err)
		return nil, err
	}
	ptrOrSchema := &PtrOrSchemaJSONDraft05{
		PtrOrSchema_JSONDraft05: &PtrOrSchemaJSONDraft05_Schema{schema},
	}
	return ptrOrSchema, nil
}

func specMaybeAddType(t Schema_JSONDraft05_Type, ts []Schema_JSONDraft05_Type) {
	for _, aT := range ts {
		if t == aT {
			return
		}
	}
	ts = append(ts, t)
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
