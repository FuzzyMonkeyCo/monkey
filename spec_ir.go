package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jban332/kin-openapi/openapi3"
)

type mapKeyToPtrOrSchema map[string]*PtrOrSchemaJSON
type mapXXXToPtrOrSchema map[uint32]*PtrOrSchemaJSON

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
			Json: schemas,
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
		var ptrOrSchema *PtrOrSchemaJSON
		if ptrOrSchema, err = specPtrOrSchemaFromDoc(ptr, schemaRef); err != nil {
			return
		}
		colorERR.Printf("%#v --> %v\n", ptr, ptrOrSchema)
		schemas[ptr] = ptrOrSchema
	}

	return
}

func specPtrOrSchemaFromDoc(ptr string, schemaRef *openapi3.SchemaRef) (
	schema *PtrOrSchemaJSON,
	err error,
) {
	//FIXME: Schemas::map[str]schemaORptr, Endpoints::only ptrs to schemas
	if schemaRef.Ref != "" {
		if schemaRef.Value == nil {
			err = fmt.Errorf("%s is neither ref nor schema", ptr)
			log.Println("[ERR]", err)
			return
		}
		schema = &PtrOrSchemaJSON{
			PtrOrSchema_JSON: &PtrOrSchemaJSON_Ptr{
				Ptr: schemaRef.Ref,
			},
		}
	}

	schema, err = specSchemaFromDocSchema(ptr, schemaRef.Value)
	return
}

func specSchemaFromDocSchema(ptr string, s *openapi3.Schema) (*PtrOrSchemaJSON, error) {
	wasSet := false
	schema := &Schema_JSON{}

	// enum
	// sEnum := s.GetEnum()
	// if len(sEnum) != 0 {
	// 	schema.Enum =
	// }

	// nullable
	if s.Nullable {
		schema.Type = []Schema_JSON_Type{Schema_JSON_null}
		wasSet = true
	}

	// type
	sType := s.Type
	if sType != "" {
		t := Schema_JSON_Type(Schema_JSON_Type_value[sType])
		specMaybeAddType(t, schema.Type)
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
		specMaybeAddType(Schema_JSON_object, schema.Type)
		schema.Required = s.Required
		schema.Properties = make(mapKeyToPtrOrSchema)
		for propName, propSchemaRef := range sProperties {
			subPtr := ptr + "/" + propName
			subS, err := specPtrOrSchemaFromDoc(subPtr, propSchemaRef)
			if err != nil {
				return nil, err
			}
			schema.Properties[propName] = subS
		}
		wasSet = true
	}

	//FIXME: support all Schema.JSON fields

	if !wasSet {
		err := fmt.Errorf("%s is an empty schema: %#v", ptr, s)
		log.Println("[ERR]", err)
		return nil, err
	}
	ptrOrSchema := &PtrOrSchemaJSON{
		PtrOrSchema_JSON: &PtrOrSchemaJSON_Schema{schema},
	}
	return ptrOrSchema, nil
}

func specMaybeAddType(t Schema_JSON_Type, ts []Schema_JSON_Type) {
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
		path := &Path{
			Partial: []*Path_PathPartial{
				{Pp: &Path_PathPartial_Part{basePath}},
				{Pp: &Path_PathPartial_Ptr{parameterizedPath[1:]}},
			},
		}

		for docMethod, docOp := range docPathItem.Operations() {
			method := Method(Method_value[docMethod])
			params, err := specEndpointParams(docOp.Parameters, docOp.RequestBody)
			if err != nil {
				return endpoints, err
			}
			outputs, err := specEndpointResponses(docOp.Responses)
			if err != nil {
				return endpoints, err
			}

			endpoint := &Endpoint{
				Endpoint: &Endpoint_Json{
					&EndpointJSON{
						Method:  method,
						Path:    path,
						Params:  params,
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

func specEndpointParams(
	docParams openapi3.Parameters,
	docReqBody *openapi3.RequestBodyRef,
) (
	params *ParamsJSON,
	err error,
) {
	type paramsJSON map[string]*ParamJSON
	params = &ParamsJSON{
		Header: make(paramsJSON),
		Path:   make(paramsJSON),
		Body:   make(paramsJSON),
		Query:  make(paramsJSON),
	}

	if docReqBody != nil {
		docBody := docReqBody.Value
		if docBody == nil {
			err = fmt.Errorf("unresolved response %#v", docReqBody)
			log.Println("[ERR]", err)
			return
		}

		for mime, ct := range docBody.Content {
			if mime == mimeJSON {
				ptr := "FIXME"
				schema, err := specPtrOrSchemaFromDoc(ptr, ct.Schema)
				if err != nil {
					return params, err
				}
				params.Body[ptr] = &ParamJSON{
					SchemaOrPtr: schema,
					Connected:   specRefConnected(ct.Schema),
					Required:    docBody.Required,
				}
			}
		}
	}

	for _, docParamRef := range docParams {
		docParam := docParamRef.Value
		if docParam == nil {
			err = fmt.Errorf("unresolved response %#v", docParamRef)
			log.Println("[ERR]", err)
			return
		}

		ptr := "#/components/parameters/" + docParam.Name
		schema, err := specPtrOrSchemaFromDoc(ptr, docParam.Schema)
		if err != nil {
			return params, err
		}
		param := &ParamJSON{
			SchemaOrPtr: schema,
			Connected:   specRefConnected(docParam.Schema),
			Required:    docParam.Required,
		}

		switch docParam.In {
		case openapi3.ParameterInPath:
			params.Path[ptr] = param
		}
	}

	return
}

func specRefConnected(schema *openapi3.SchemaRef) bool {
	return schema.Ref != ""
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
		if responseRef.Value == nil {
			err = fmt.Errorf("unresolved response %#v", responseRef)
			log.Println("[ERR]", err)
			return outputs, err
		}
		for mime, ct := range responseRef.Value.Content {
			if mime == mimeJSON {
				schema, err := specPtrOrSchemaFromDoc("", ct.Schema)
				if err != nil {
					return outputs, err
				}
				outputs[xxx] = schema
			}
		}
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
