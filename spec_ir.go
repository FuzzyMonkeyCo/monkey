package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jban332/kin-openapi/openapi3"
)

type mapKeyToPtrOrSchema map[string]*PtrOrSchemaJSONDraft05
type mapXXXToPtrOrSchema map[uint32]*PtrOrSchemaJSONDraft05

func newSpecFromOpenAPIv3(doc *openapi3.Swagger) (spec *SpecIR, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	schemas, err := specSchemas("#/components/schemas/", doc.Components.Schemas)
	if err != nil {
		return
	}

	basePath, err := specBasePath(doc.Servers)
	if err != nil {
		return
	}
	endpoints, err := specEndpoints(basePath, doc.Paths)
	if err != nil {
		return
	}

	spec = &SpecIR{
		Endpoints: endpoints,
		Schemas: &Schemas{
			Schemas_JSONDraft05: schemas,
		},
	}
	log.Printf("\n basePath:%#v\n spec: %v\n ", basePath, spec)

	stringified, err := new(jsonpb.Marshaler).MarshalToString(spec)
	log.Println("[DBG]", err, stringified)
	return
}

func specSchemas(baseRef string, docSchemas map[string]*openapi3.SchemaRef) (
	schemas mapKeyToPtrOrSchema,
	err error,
) {
	schemas = make(mapKeyToPtrOrSchema)

	for name, schemaRef := range docSchemas {
		ptr := baseRef + name
		if schemaRef.Ref != "" {
			if schemaRef.Value == nil {
				err = newErrorNeitherRefNorSchema(ptr)
				log.Println("[ERR]", err)
				return
			}
			schemas[ptr] = &PtrOrSchemaJSONDraft05{
				PtrOrSchema_JSONDraft05: &PtrOrSchemaJSONDraft05_Ptr{Ptr: schemaRef.Ref},
			}
		}

		var ptrOrSchema *PtrOrSchemaJSONDraft05
		if ptrOrSchema, err = specSchemaFromDocSchema(ptr, schemaRef.Value); err != nil {
			return
		}
		colorERR.Printf("%#v --> %v\n", ptr, ptrOrSchema)
		schemas[ptr] = ptrOrSchema
	}

	return
}

func newErrorNeitherRefNorSchema(ptr string) error {
	return fmt.Errorf("%s is neither ref nor schema", ptr)
}

func specSchemaFromDocSchema(ptr string, s *openapi3.Schema) (*PtrOrSchemaJSONDraft05, error) {
	wasSet := false
	schema := &Schema_JSONDraft05{}

	// enum
	// sEnum := s.GetEnum()
	// if len(sEnum) != 0 {
	// 	schema.Enum =
	// }

	// type, nullable
	sType := s.Type
	if sType != "" {
		t := Schema_JSONDraft05_Type(Schema_JSONDraft05_Type_value[sType])
		if s.Nullable {
			schema.Type = []Schema_JSONDraft05_Type{t, Schema_JSONDraft05_null}
		} else {
			schema.Type = []Schema_JSONDraft05_Type{t}
		}
		wasSet = true
	}

	// format
	sFormat := s.Format
	if sFormat != "" {
		schema.Format = sFormat
		wasSet = true
	}

	// properties, required
	sProperties := s.Properties
	if len(sProperties) != 0 {
		specMaybeAddType(Schema_JSONDraft05_object, schema.Type)
		schema.Required = s.Required
		schema.Properties = make(mapKeyToPtrOrSchema)
		for propName, propSchemaRef := range sProperties {
			subPtr := ptr + "/" + propName
			if propSchemaRef.Ref != "" {
				if propSchemaRef.Value == nil {
					err := newErrorNeitherRefNorSchema(subPtr)
					log.Println("[ERR]", err)
					return nil, err
				}
				err := fmt.Errorf("%s is a ref and that's not yet supported", subPtr)
				log.Println("[ERR]", err)
				log.Printf("[ERR] >>> %#v\n", propSchemaRef)
				return nil, err
			}

			subS, err := specSchemaFromDocSchema(subPtr, propSchemaRef.Value)
			if err != nil {
				return nil, err
			}
			schema.Properties[propName] = subS
		}
		wasSet = true
	}

	//FIXME: support all Schema.JSONDraft05 fields

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

func specEndpoints(basePath string, docPaths openapi3.Paths) (
	endpoints []*Endpoint,
	err error,
) {
	for parameterizedPath, docPathItem := range docPaths {
		partials := &Path{
			Partial: []*Path_PathPartial{
				{Pp: &Path_PathPartial_Part{basePath}},
				{Pp: &Path_PathPartial_Ptr{parameterizedPath[1:]}},
			},
		}

		for docMethod, docOp := range docPathItem.Operations() {
			method := Method(Method_value[docMethod])
			outputs, err := specEndpointResponses(docOp.Responses)
			if err != nil {
				return endpoints, err
			}

			endpoint := &Endpoint{
				Endpoint: &Endpoint_Endpoint_JSONDraft05{
					&EndpointJSONDraft05{
						Method:  method,
						Path:    partials,
						Params:  &ParamsJSONDraft05{},
						Outputs: outputs,
					},
				},
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	return
}

func specXXX(code string) (xxx uint32, err error) {
	var i int
	switch {
	case code == "default":
		xxx = 0
	case code == "1XX":
		xxx = 1
	case code == "2XX":
		xxx = 2
	case code == "3XX":
		xxx = 3
	case code == "4XX":
		xxx = 4
	case code == "5XX":
		xxx = 5

	case "100" <= code && code <= "199":
		i, err = strconv.Atoi(code)
		xxx = uint32(i)
	case "200" <= code && code <= "299":
		i, err = strconv.Atoi(code)
		xxx = uint32(i)
	case "300" <= code && code <= "399":
		i, err = strconv.Atoi(code)
		xxx = uint32(i)
	case "400" <= code && code <= "499":
		i, err = strconv.Atoi(code)
		xxx = uint32(i)
	case "500" <= code && code <= "599":
		i, err = strconv.Atoi(code)
		xxx = uint32(i)

	default:
		err = fmt.Errorf("unexpected output HTTP code: '%s'", code)
		log.Println("[ERR]", err)
	}
	return
}

func specEndpointResponses(docResponses openapi3.Responses) (
	outputs mapXXXToPtrOrSchema,
	err error,
) {
	outputs = make(mapXXXToPtrOrSchema)

	for code, responseRef := range docResponses {
		xxx, err := specXXX(code)
		if err != nil {
			return outputs, err
		}
		log.Println("xxx =", xxx)
		if responseRef.Value == nil {
			err = fmt.Errorf("unresolved response %#v", responseRef)
			log.Println("[ERR]", err)
			return outputs, err
		}
		// for mime, ct := range responseRef.Value.Content {
		// 	if mimeJSON == mime {
		// 		schema, err := specSchemaFromDocSchema("", ct.Schema)
		// 	}
		// }
	}

	return
}

//TODO: support the whole spec on /"servers"
func specBasePath(docServers openapi3.Servers) (
	basePath string,
	err error,
) {
	if len(docServers) == 0 {
		log.Println(`[NFO] field 'servers' empty/unset: using "/"`)
		basePath = "/"
		return
	}

	if len(docServers) != 1 {
		log.Println(`[NFO] field 'servers' has many values: using the first one`)
	}
	basePath = docServers[0].URL
	if basePath == "" || basePath[0] != '/' {
		err = errors.New(`field 'servers' has no suitable 'url'`)
		log.Println("[ERR]", err)
		colorERR.Println(err)
	}
	return
}
