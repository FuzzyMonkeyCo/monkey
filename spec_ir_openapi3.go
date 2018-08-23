package main

import (
	"errors"
	"log"
	"net/url"
	"reflect"
	"sort"
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
			if s := schPtr.GetSchema(); s != nil && reflect.DeepEqual(schema, s) {
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

func (sm schemap) schemasFromOA3(baseRef string, docSchemas map[string]*openapi3.SchemaRef) {
	i, names := 0, make([]string, len(docSchemas))
	for name := range docSchemas {
		names[i] = name
		i++
	}
	sort.Strings(names)

	for j := 0; j != i; j++ {
		name := names[j]
		schema := sm.schemaFromOA3(docSchemas[name].Value)
		sm.seed(baseRef+name, schema)
	}
}

func (sm schemap) endpointsFromOA3(basePath string, docPaths openapi3.Paths) (
	endpoints []*Endpoint,
) {
	i, paths := 0, make([]string, len(docPaths))
	for path := range docPaths {
		paths[i] = path
		i++
	}
	sort.Strings(paths)

	for j := 0; j != i; j++ {
		path := paths[j]
		partials := pathFromOA3(basePath, path)
		docOps := docPaths[path].Operations()
		k, methods := 0, make([]string, len(docOps))
		for docMethod := range docOps {
			methods[k] = docMethod
			k++
		}
		sort.Strings(methods)

		for l := 0; l != k; l++ {
			docMethod := methods[l]
			docOp := docOps[docMethod]
			inputs := make([]*ParamJSON, 0, 1+len(docOp.Parameters))
			sm.inputBodyFromOA3(&inputs, docOp.RequestBody)
			sm.inputsFromOA3(&inputs, docOp.Parameters)
			outputs := sm.outputsFromOA3(docOp.Responses)
			endpoint := &Endpoint{
				Endpoint: &Endpoint_Json{
					&EndpointJSON{
						Method:       Method(Method_value[docMethod]),
						PathPartials: partials,
						Inputs:       inputs,
						Outputs:      outputs,
					},
				},
			}
			endpoints = append(endpoints, endpoint)
		}
	}
	return
}

func (sm schemap) inputBodyFromOA3(inputs *[]*ParamJSON, docReqBody *openapi3.RequestBodyRef) {
	if docReqBody != nil {
		//FIXME: handle .Ref
		docBody := docReqBody.Value
		for mime, ct := range docBody.Content {
			if mime == mimeJSON {
				param := &ParamJSON{
					Required: docBody.Required,
					Ptr:      sm.ensureMappedOA3SchemaRef(ct.Schema),
					Name:     "",
					Kind:     ParamJSON_body,
				}
				*inputs = append(*inputs, param)
				return
			}
		}
	}
}

func (sm schemap) inputsFromOA3(inputs *[]*ParamJSON, docParams openapi3.Parameters) {
	paramsCount := len(docParams)
	paramap := make(map[string]*openapi3.ParameterRef, paramsCount)
	i, names := 0, make([]string, paramsCount)
	for _, docParamRef := range docParams {
		docParam := docParamRef.Value
		name := docParam.In + docParam.Name
		names[i] = name
		paramap[name] = docParamRef
		i++
	}
	sort.Strings(names)

	for j := 0; j != i; j++ {
		docParamRef := paramap[names[j]]
		//FIXME: handle .Ref
		docParam := docParamRef.Value
		kind := ParamJSON_UNKNOWN
		switch docParam.In {
		case openapi3.ParameterInPath:
			kind = ParamJSON_path
		case openapi3.ParameterInQuery:
			kind = ParamJSON_query
		case openapi3.ParameterInHeader:
			kind = ParamJSON_header
		case openapi3.ParameterInCookie:
			kind = ParamJSON_cookie
		}
		param := &ParamJSON{
			Required: docParam.Required,
			Ptr:      sm.ensureMappedOA3SchemaRef(docParam.Schema),
			Name:     docParam.Name,
			Kind:     kind,
		}
		*inputs = append(*inputs, param)
	}
}

func (sm schemap) outputsFromOA3(docResponses openapi3.Responses) (
	outputs map[uint32]*SchemaPtr,
) {
	outputs = make(map[uint32]*SchemaPtr)
	for code, responseRef := range docResponses {
		//FIXME: handle .Ref
		for mime, ct := range responseRef.Value.Content {
			if mime == mimeJSON {
				xxx := makeXXXFromOA3(code)
				outputs[xxx] = sm.ensureMappedOA3SchemaRef(ct.Schema)
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
	schema.Format = formatFromOA3(s.Format)
	// "minLength"
	schema.MinLength = s.MinLength
	// "maxLength"
	if sMaxLength := s.MaxLength; nil != sMaxLength {
		schema.MaxLength = *sMaxLength
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
		if sItems.Value.IsEmpty() {
			schema.Items = []*SchemaPtr{}
		} else {
			schemaPtr := sm.ensureMappedOA3SchemaRef(sItems)
			schema.Items = []*SchemaPtr{schemaPtr}
		}
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
	if count := len(s.Properties); count != 0 {
		ensureSchemaType(Schema_JSON_object, &schema.Type)
		schema.Properties = make(map[string]*SchemaPtr, count)
		i, props := 0, make([]string, count)
		for propName := range s.Properties {
			props[i] = propName
			i++
		}
		sort.Strings(props)

		for j := 0; j != i; j++ {
			propName := props[j]
			schemaPtr := sm.ensureMappedOA3SchemaRef(s.Properties[propName])
			schema.Properties[propName] = schemaPtr
		}
	}
	//FIXME: "additionalProperties"

	// "allOf"
	if sAllOf := s.AllOf; len(sAllOf) != 0 {
		schema.AllOf = make([]*SchemaPtr, len(sAllOf))
		for i, sOf := range sAllOf {
			schema.AllOf[i] = sm.ensureMappedOA3SchemaRef(sOf)
		}
	}

	// "anyOf"
	if sAnyOf := s.AnyOf; len(sAnyOf) != 0 {
		schema.AnyOf = make([]*SchemaPtr, len(sAnyOf))
		for i, sOf := range sAnyOf {
			schema.AnyOf[i] = sm.ensureMappedOA3SchemaRef(sOf)
		}
	}

	// "oneOf"
	if sOneOf := s.OneOf; len(sOneOf) != 0 {
		schema.OneOf = make([]*SchemaPtr, len(sOneOf))
		for i, sOf := range sOneOf {
			schema.OneOf[i] = sm.ensureMappedOA3SchemaRef(sOf)
		}
	}

	// "not"
	if sNot := s.Not; nil != sNot {
		schema.Not = sm.ensureMappedOA3SchemaRef(sNot)
	}

	return
}

func formatFromOA3(format string) Schema_JSON_Format {
	switch format {
	case "date-time":
		return Schema_JSON_date_time
	case "uriref", "uri-reference":
		return Schema_JSON_uri_reference
	default:
		v, ok := Schema_JSON_Format_value[format]
		if ok {
			return Schema_JSON_Format(v)
		}
		return Schema_JSON_NONE
	}
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
	sm.schemasFromOA3("#/components/schemas/", docSchemas)

	basePath, err := basePathFromOA3(doc.Servers)
	if err != nil {
		return
	}
	spec = &SpecIR{
		Endpoints: sm.endpointsFromOA3(basePath, doc.Paths),
		Schemas:   &Schemas{Json: sm},
	}

	jsoner := &jsonpb.Marshaler{
		// Indent: "\t",
		// EmitDefaults: true,
	}
	stringified, err := jsoner.MarshalToString(spec)
	log.Println("[DBG]", err, stringified)
	return
}

func pathFromOA3(basePath, path string) (partials []*PathPartial) {
	if basePath != "/" {
		p := &PathPartial{Pp: &PathPartial_Part{basePath}}
		partials = append(partials, p)
	}

	onCurly := func(r rune) bool { return r == '{' || r == '}' }
	isCurly := '{' == path[0]
	for i, part := range strings.FieldsFunc(path, onCurly) {
		var p PathPartial
		if isCurly || i%2 != 0 {
			p.Pp = &PathPartial_Ptr{part}
		} else {
			p.Pp = &PathPartial_Part{part}
		}
		partials = append(partials, &p)
	}

	if length := len(partials); length > 1 {
		part1 := partials[0].GetPart()
		part2 := partials[1].GetPart()
		if part1 != "" && part2 != "" {
			partials = partials[1:]
			partials[0] = &PathPartial{Pp: &PathPartial_Part{part1 + part2}}
			return
		}
	}
	return
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
