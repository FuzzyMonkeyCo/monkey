package main

import (
	"errors"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golang/protobuf/jsonpb"
)

type schemap map[uint32]*RefOrSchemaJSON

func newSchemap(capa int) schemap {
	return make(schemap, capa)
}

func (sm schemap) newUID() uint32 {
	return uint32(1 + len(sm))
}

func (sm schemap) seed(ref string, schema *Schema_JSON) {
	schemaPtr := sm.ensureMapped("", schema)
	schemaPtr.Ref = ref
	sm[sm.newUID()] = &RefOrSchemaJSON{
		PtrOrSchema: &RefOrSchemaJSON_Ptr{schemaPtr},
	}
}

func (sm schemap) ensureMapped(ref string, schema *Schema_JSON) *SchemaPtr {
	if ref == "" {
		for UID, schPtr := range sm {
			//TODO: try loop & search for schema to save on space
			if s := schPtr.GetSchema(); s != nil && schema.equal(s) {
				log.Printf(">>> yay!")
				return &SchemaPtr{UID: UID}
			}
		}
		UID := sm.newUID()
		sm[UID] = &RefOrSchemaJSON{
			PtrOrSchema: &RefOrSchemaJSON_Schema{schema},
		}
		return &SchemaPtr{UID: UID}
	}

	if schema == nil {
		panic("no ref nor schema!")
	}

	mappedUID := uint32(0)
	for UID, schPtr := range sm {
		if ptr := schPtr.GetPtr(); ptr != nil && ref == ptr.GetRef() {
			mappedUID = UID
			break
		}
	}
	schemaPtr := &SchemaPtr{
		Ref: ref,
		UID: mappedUID,
	}
	sm[sm.newUID()] = &RefOrSchemaJSON{
		PtrOrSchema: &RefOrSchemaJSON_Ptr{schemaPtr},
	}
	return schemaPtr
}

func (sm schemap) ensureMappedOA3SchemaRef(s *openapi3.SchemaRef) *SchemaPtr {
	if docSchema := s.Value; docSchema != nil {
		schema := sm.schemaFromOA3(docSchema)
		return sm.ensureMapped("", schema)
	}
	if s.Ref != "" {
		return sm.ensureMapped(s.Ref, nil)
	}
	panic("both schema and ref are empty")
}

func (sm schemap) addOA3Schemas(baseRef string, docSchemas map[string]*openapi3.SchemaRef) {
	for name, schemaRef := range docSchemas {
		ref := baseRef + name
		colorERR.Printf(">>> %#v\n", ref)
		schema := sm.schemaFromOA3(schemaRef.Value)
		colorERR.Printf(">>> %#v --> %#v\n", schemaRef.Value, schema)
		sm.seed(ref, schema)
	}
}

func (sm schemap) endpointsFromOA3(basePath string, docPaths openapi3.Paths) (
	endpoints []*Endpoint,
) {
	for parameterizedPath, docPathItem := range docPaths {
		path := pathFromOA3(basePath, parameterizedPath)

		for docMethod, docOp := range docPathItem.Operations() {
			method := Method(Method_value[docMethod])
			inputs := sm.inputsFromOA3(docOp.Parameters, docOp.RequestBody)
			outputs := sm.outputsFromOA3(docOp.Responses)

			endpoint := &Endpoint{
				Endpoint: &Endpoint_Json{
					&EndpointJSON{
						Method:  method,
						Path:    path,
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
			}
			endpoints = append(endpoints, endpoint)
		}
	}
	return
}

func (sm schemap) inputsFromOA3(
	docParams openapi3.Parameters,
	docReqBody *openapi3.RequestBodyRef,
) (
	params *ParamsJSON,
) {
	type paramsJSON map[string]*ParamJSON
	params = &ParamsJSON{
		Header: make(paramsJSON),
		Path:   make(paramsJSON),
		Query:  make(paramsJSON),
	}

	if docReqBody != nil {
		//FIXME: handle .Ref
		docBody := docReqBody.Value
		for mime, ct := range docBody.Content {
			if mime == mimeJSON {
				params.Body = &ParamJSON{
					Required: docBody.Required,
					Ptr:      sm.ensureMappedOA3SchemaRef(ct.Schema),
				}
			}
		}
	}

	for _, docParamRef := range docParams {
		//FIXME: handle .Ref
		docParam := docParamRef.Value
		param := &ParamJSON{
			Required: docParam.Required,
			Ptr:      sm.ensureMappedOA3SchemaRef(docParam.Schema),
		}

		switch docParam.In {
		case openapi3.ParameterInPath:
			params.Path[docParam.Name] = param
		}
	}
	return
}

func (sm schemap) outputsFromOA3(docResponses openapi3.Responses) (
	outputs map[uint32]*SchemaPtr,
) {
	outputs = make(map[uint32]*SchemaPtr)
	for code, responseRef := range docResponses {
		//FIXME: handle .Ref
		for mime, ct := range responseRef.Value.Content {
			if mime == mimeJSON {
				schemaPtr := sm.ensureMappedOA3SchemaRef(ct.Schema)
				outputs[makeXXXFromOA3(code)] = schemaPtr
			}
		}
	}
	return
}

func (sm schemap) schemaFromOA3(s *openapi3.Schema) (schema *Schema_JSON) {
	schema = &Schema_JSON{}
	// "enum" FIXME

	// "nullable"
	if s.Nullable {
		schema.Type = []Schema_JSON_Type{Schema_JSON_null}
	}
	// "type"
	if sType := s.Type; sType != "" {
		t := Schema_JSON_Type(Schema_JSON_Type_value[sType])
		ensureSchemaType(t, &schema.Type)
	}

	// "format"
	schema.Format = s.Format
	// "minLength"
	schema.MinLength = s.MinLength
	// "maxLength"
	if nil != s.MaxLength {
		schema.MaxLength = *s.MaxLength
		schema.HasMaxLength = true
	}
	// "pattern"
	schema.Pattern = s.Pattern

	// "minimum"
	if nil != s.Min {
		schema.Minimum = *s.Min
		schema.HasMinimum = true
	}
	// "maximum"
	if nil != s.Max {
		schema.Maximum = *s.Max
		schema.HasMaximum = true
	}
	// "exclusiveMinimum", "exclusiveMaximum"
	schema.ExclusiveMinimum = s.ExclusiveMin
	schema.ExclusiveMaximum = s.ExclusiveMax
	// "multipleOf"
	if nil != s.MultipleOf {
		schema.TranslatedMultipleOf = *s.MultipleOf - 1.0
	}

	// "uniqueItems"
	schema.UniqueItems = s.UniqueItems
	// "minItems"
	schema.MinItems = s.MinItems
	// "maxItems"
	if nil != s.MaxItems {
		schema.MaxItems = *s.MaxItems
		schema.HasMaxItems = true
	}
	// "items"
	if sItems := s.Items; nil != sItems {
		ensureSchemaType(Schema_JSON_array, &schema.Type)
		schemaPtr := sm.ensureMappedOA3SchemaRef(sItems)
		schema.Items = []*SchemaPtr{schemaPtr}
	}

	// "minProperties"
	schema.MinProperties = s.MinProps
	// "maxProperties"
	if nil != s.MaxProps {
		schema.MaxProperties = *s.MaxProps
		schema.HasMaxProperties = true
	}
	// "required"
	schema.Required = s.Required
	// "properties"
	if sProperties := s.Properties; len(sProperties) != 0 {
		ensureSchemaType(Schema_JSON_object, &schema.Type)
		schema.Properties = make(map[string]*SchemaPtr, len(sProperties))
		for propName, propSchemaRef := range sProperties {
			schemaPtr := sm.ensureMappedOA3SchemaRef(propSchemaRef)
			schema.Properties[propName] = schemaPtr
		}
	}
	//FIXME: "additionalProperties"

	// "allOf"
	if sAllOf := s.AllOf; len(sAllOf) != 0 {
		schema.AllOf = make([]*SchemaPtr, len(sAllOf))
		for i, sOf := range sAllOf {
			schemaPtr := sm.ensureMappedOA3SchemaRef(sOf)
			schema.AllOf[i] = schemaPtr
		}
	}

	// "anyOf"
	if sAnyOf := s.AnyOf; len(sAnyOf) != 0 {
		schema.AnyOf = make([]*SchemaPtr, len(sAnyOf))
		for i, sOf := range sAnyOf {
			schemaPtr := sm.ensureMappedOA3SchemaRef(sOf)
			schema.AnyOf[i] = schemaPtr
		}
	}

	// "oneOf"
	if sOneOf := s.OneOf; len(sOneOf) != 0 {
		schema.OneOf = make([]*SchemaPtr, len(sOneOf))
		for i, sOf := range sOneOf {
			schemaPtr := sm.ensureMappedOA3SchemaRef(sOf)
			schema.OneOf[i] = schemaPtr
		}
	}

	// "not"
	if sNot := s.Not; nil != sNot {
		schemaPtr := sm.ensureMappedOA3SchemaRef(sNot)
		schema.Not = schemaPtr
	}

	return
}

func (s *Schema_JSON) equal(ss *Schema_JSON) bool {
	return reflect.DeepEqual(s, ss)
}

func ensureSchemaType(t Schema_JSON_Type, ts *[]Schema_JSON_Type) {
	for _, aT := range *ts {
		if t == aT {
			return
		}
	}
	*ts = append(*ts, t)
}

// https://swagger.io/docs/specification/data-models/data-types/
func newSpecFromOA3(doc *openapi3.Swagger) (spec *SpecIR, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	docSchemas := doc.Components.Schemas
	sm := newSchemap(len(docSchemas))
	sm.addOA3Schemas("#/components/schemas/", docSchemas)

	basePath, err := basePathFromOA3(doc.Servers)
	if err != nil {
		return
	}
	endpoints := sm.endpointsFromOA3(basePath, doc.Paths)

	spec = &SpecIR{
		Endpoints: endpoints,
		Schemas:   &Schemas{Json: sm},
	}
	log.Printf("\n basePath:%#v\n spec: %v\n ", basePath, spec)

	stringified, err := new(jsonpb.Marshaler).MarshalToString(spec)
	log.Println("[DBG]", err, stringified)
	return
}

func pathFromOA3(basePath, parameterizedPath string) *Path {
	var partials []*Path_PathPartial
	if basePath != "/" {
		p := &Path_PathPartial{Pp: &Path_PathPartial_Part{basePath}}
		partials = append(partials, p)
	}
	onCurly := func(r rune) bool { return r == '{' || r == '}' }
	isCurly := '{' == parameterizedPath[0]
	for i, part := range strings.FieldsFunc(parameterizedPath, onCurly) {
		var p Path_PathPartial
		if isCurly || i%2 != 0 {
			p.Pp = &Path_PathPartial_Ptr{part}
		} else {
			p.Pp = &Path_PathPartial_Part{part}
		}
		partials = append(partials, &p)
	}
	return &Path{Partial: partials}
}

func makeXXXFromOA3(code string) uint32 {
	switch {
	case code == "default":
		return 0
	case code == "1XX":
		return 1
	case code == "2XX":
		return 2
	case code == "3XX":
		return 3
	case code == "4XX":
		return 4
	case code == "5XX":
		return 5

	case "100" <= code && code <= "199":
		i, _ := strconv.Atoi(code)
		return uint32(i)
	case "200" <= code && code <= "299":
		i, _ := strconv.Atoi(code)
		return uint32(i)
	case "300" <= code && code <= "399":
		i, _ := strconv.Atoi(code)
		return uint32(i)
	case "400" <= code && code <= "499":
		i, _ := strconv.Atoi(code)
		return uint32(i)
	case "500" <= code && code <= "599":
		i, _ := strconv.Atoi(code)
		return uint32(i)

	default:
		panic(code)
	}
}

//TODO: support the whole spec on /"servers"
func basePathFromOA3(docServers openapi3.Servers) (basePath string, err error) {
	if len(docServers) == 0 {
		log.Println(`[NFO] field 'servers' empty/unset: using "/"`)
		basePath = "/"
		return
	}

	if len(docServers) != 1 {
		log.Println(`[NFO] field 'servers' has many values: using the first one`)
	}

	u, err := url.Parse(docServers[0].URL)
	if err != nil {
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}
	basePath = u.Path

	if basePath == "" || basePath[0] != '/' {
		err = errors.New(`field 'servers' has no suitable 'url'`)
		log.Println("[ERR]", err)
		colorERR.Println(err)
	}
	return
}
