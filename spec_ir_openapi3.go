package main

import (
	"errors"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golang/protobuf/jsonpb"
)

func newSpecFromOA3(doc *openapi3.Swagger) (val *validator, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	docPaths, docSchemas := doc.Paths, doc.Components.Schemas
	vald := newValidator(len(docPaths), len(docSchemas))
	log.Println("[DBG] seeding schemas")
	//TODO: use docPath as root
	vald.schemasFromOA3(docSchemas)

	basePath, err := basePathFromOA3(doc.Servers)
	if err != nil {
		return
	}
	log.Println("[DBG] going through endpoints")
	vald.endpointsFromOA3(basePath, docPaths)

	log.Println("[DBG] serializing the protobuf")
	jsoner := &jsonpb.Marshaler{
		// Indent: "\t",
		// EmitDefaults: true,
	}
	stringified, err := jsoner.MarshalToString(vald.Spec)
	log.Println("[DBG]", err, stringified[:37])
	return
}

func (vald *validator) schemasFromOA3(docSchemas map[string]*openapi3.SchemaRef) error {
	schemas := make(schemasJSON, len(docSchemas))
	for name, docSchema := range docSchemas {
		schemas[name] = vald.schemaFromOA3(docSchema.Value)
	}
	return vald.seed("#/components/schemas/", schemas)
}

func (sm schemap) ensureMappedOA3SchemaRef(s *openapi3.SchemaRef) sid {
	if docSchema := s.Value; docSchema != nil {
		schema := sm.schemaFromOA3(docSchema)
		return sm.ensureMapped("", schema)
	}
	// if s.Ref != "" {
	// 	return sm.ensureMapped(s.Ref, nil)
	// }
	panic("both schema and ref are empty")
}

func (vald *validator) endpointsFromOA3(basePath string, docPaths openapi3.Paths) {
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
			vald.Spec.Endpoints = append(vald.Spec.Endpoints, endpoint)
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
					SID:      sm.ensureMappedOA3SchemaRef(ct.Schema),
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
			SID:      sm.ensureMappedOA3SchemaRef(docParam.Schema),
			Name:     docParam.Name,
			Kind:     kind,
		}
		*inputs = append(*inputs, param)
	}
}

func (sm *schemap) outputsFromOA3(docResponses openapi3.Responses) (
	outputs map[uint32]sid,
) {
	outputs = make(map[uint32]sid)
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

func (sm schemap) schemaFromOA3(s *openapi3.Schema) (schema Schema_JSON) {
	// "enum"
	if sEnum := s.Enum; len(sEnum) != 0 {
		schema.Enum = make([]*ValueJSON, len(sEnum))
		for i, v := range sEnum {
			schema.Enum[i] = enumFromOA3(v)
		}
	}

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
			schema.Items = []sid{}
		} else {
			SID := sm.ensureMappedOA3SchemaRef(sItems)
			schema.Items = []sid{SID}
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
		schema.Properties = make(map[string]sid, count)
		i, props := 0, make([]string, count)
		for propName := range s.Properties {
			props[i] = propName
			i++
		}
		sort.Strings(props)

		for j := 0; j != i; j++ {
			propName := props[j]
			SID := sm.ensureMappedOA3SchemaRef(s.Properties[propName])
			schema.Properties[propName] = SID
		}
	}
	//FIXME: "additionalProperties"

	// "allOf"
	if sAllOf := s.AllOf; len(sAllOf) != 0 {
		schema.AllOf = make([]sid, len(sAllOf))
		for i, sOf := range sAllOf {
			schema.AllOf[i] = sm.ensureMappedOA3SchemaRef(sOf)
		}
	}

	// "anyOf"
	if sAnyOf := s.AnyOf; len(sAnyOf) != 0 {
		schema.AnyOf = make([]sid, len(sAnyOf))
		for i, sOf := range sAnyOf {
			schema.AnyOf[i] = sm.ensureMappedOA3SchemaRef(sOf)
		}
	}

	// "oneOf"
	if sOneOf := s.OneOf; len(sOneOf) != 0 {
		schema.OneOf = make([]sid, len(sOneOf))
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

func enumFromOA3(value interface{}) *ValueJSON {
	if value == nil {
		return &ValueJSON{Value: &ValueJSON_IsNull{true}}
	}
	switch value.(type) {
	case bool:
		return &ValueJSON{Value: &ValueJSON_Boolean{value.(bool)}}
	case float64:
		return &ValueJSON{Value: &ValueJSON_Number{value.(float64)}}
	case string:
		return &ValueJSON{Value: &ValueJSON_Text{value.(string)}}
	case []interface{}:
		val := value.([]interface{})
		vs := make([]*ValueJSON, len(val))
		for i, v := range val {
			vs[i] = enumFromOA3(v)
		}
		return &ValueJSON{Value: &ValueJSON_Array{&ArrayJSON{Values: vs}}}
	case map[string]interface{}:
		val := value.(map[string]interface{})
		vs := make(map[string]*ValueJSON, len(val))
		for n, v := range val {
			vs[n] = enumFromOA3(v)
		}
		return &ValueJSON{Value: &ValueJSON_Object{&ObjectJSON{Values: vs}}}
	default:
		panic("unreachable")
	}
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
