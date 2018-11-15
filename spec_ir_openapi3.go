package main

import (
	"errors"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func newSpecFromOA3(doc *openapi3.Swagger) (vald *validator, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	docPaths, docSchemas := doc.Paths, doc.Components.Schemas
	vald = newValidator(len(docPaths), len(docSchemas))
	log.Println("[DBG] seeding schemas")
	//TODO: use docPath as root of base
	vald.schemasFromOA3(docSchemas)

	basePath, err := basePathFromOA3(doc.Servers)
	if err != nil {
		return
	}
	log.Println("[DBG] going through endpoints")
	vald.endpointsFromOA3(basePath, docPaths)
	return
}

func (vald *validator) schemasFromOA3(docSchemas map[string]*openapi3.SchemaRef) error {
	schemas := make(schemasJSON, len(docSchemas))
	for name, docSchema := range docSchemas {
		schemas[name] = vald.schemaFromOA3(docSchema.Value)
	}
	return vald.seed("#/components/schemas/", schemas)
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
			vald.inputBodyFromOA3(&inputs, docOp.RequestBody)
			vald.inputsFromOA3(&inputs, docOp.Parameters)
			partials := pathFromOA3(inputs, basePath, path)
			outputs := vald.outputsFromOA3(docOp.Responses)
			endpoint := &Endpoint{
				Endpoint: &Endpoint_Json{
					&EndpointJSON{
						Method:       EndpointJSON_Method(EndpointJSON_Method_value[docMethod]),
						PathPartials: partials,
						Inputs:       inputs,
						Outputs:      outputs,
					},
				},
			}
			vald.Spec.Endpoints = append(vald.Spec.Endpoints, endpoint)
		}
	}
}

func (vald *validator) inputBodyFromOA3(inputs *[]*ParamJSON, docReqBody *openapi3.RequestBodyRef) {
	if docReqBody != nil {
		//FIXME: handle .Ref
		docBody := docReqBody.Value
		for mime, ct := range docBody.Content {
			if mime == mimeJSON {
				docSchema := ct.Schema
				schema := vald.schemaOrRefFromOA3(docSchema)
				param := &ParamJSON{
					Required: docBody.Required,
					SID:      vald.ensureMapped(docSchema.Ref, schema),
					Name:     "",
					Kind:     ParamJSON_body,
				}
				*inputs = append(*inputs, param)
				return
			}
		}
	}
}

func (vald *validator) inputsFromOA3(inputs *[]*ParamJSON, docParams openapi3.Parameters) {
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
		docSchema := docParam.Schema
		schema := vald.schemaOrRefFromOA3(docSchema)
		param := &ParamJSON{
			Required: docParam.Required,
			SID:      vald.ensureMapped(docSchema.Ref, schema),
			Name:     docParam.Name,
			Kind:     kind,
		}
		*inputs = append(*inputs, param)
	}
}

func (vald *validator) outputsFromOA3(docResponses openapi3.Responses) (
	outputs map[uint32]sid,
) {
	outputs = make(map[uint32]sid)
	i, codes := 0, make([]string, len(docResponses))
	for code := range docResponses {
		codes[i] = code
		i++
	}
	sort.Strings(codes)

	for j := 0; j != i; j++ {
		code := codes[j]
		responseRef := docResponses[code]
		//FIXME: handle .Ref
		for mime, ct := range responseRef.Value.Content {
			if mime == mimeJSON {
				xxx := makeXXXFromOA3(code)
				docSchema := ct.Schema
				schema := vald.schemaOrRefFromOA3(docSchema)
				outputs[xxx] = vald.ensureMapped(docSchema.Ref, schema)
			}
		}
	}
	return
}

func (vald *validator) schemaOrRefFromOA3(s *openapi3.SchemaRef) (schema schemaJSON) {
	if ref := s.Ref; ref != "" {
		return schemaJSON{"$ref": ref}
	}
	return vald.schemaFromOA3(s.Value)
}

func (vald *validator) schemaFromOA3(s *openapi3.Schema) (schema schemaJSON) {
	schema = make(schemaJSON)

	// "enum"
	if sEnum := s.Enum; len(sEnum) != 0 {
		schema["enum"] = sEnum
	}

	// "nullable"
	if s.Nullable {
		schema["type"] = []string{"null"}
	}
	// "type"
	if sType := s.Type; sType != "" {
		schema["type"] = ensureSchemaType(schema["type"], sType)
	}

	// "format"
	if sFormat := s.Format; sFormat != "" {
		schema["format"] = sFormat
	}
	// "minLength"
	if sMinLength := s.MinLength; sMinLength != 0 {
		schema["minLength"] = sMinLength
	}
	// "maxLength"
	if sMaxLength := s.MaxLength; nil != sMaxLength {
		schema["maxLength"] = *sMaxLength
	}
	// "pattern"
	if sPattern := s.Pattern; sPattern != "" {
		schema["pattern"] = sPattern
	}

	// "minimum"
	if nil != s.Min {
		schema["minimum"] = *s.Min
	}
	// "maximum"
	if nil != s.Max {
		schema["maximum"] = *s.Max
	}
	// "exclusiveMinimum", "exclusiveMaximum"
	if sExMin := s.ExclusiveMin; sExMin {
		schema["exclusiveMinimum"] = sExMin
	}
	if sExMax := s.ExclusiveMax; sExMax {
		schema["exclusiveMaximum"] = sExMax
	}
	// "multipleOf"
	if nil != s.MultipleOf {
		schema["multipleOf"] = *s.MultipleOf
	}

	// "uniqueItems"
	if sUniq := s.UniqueItems; sUniq {
		schema["uniqueItems"] = sUniq
	}
	// "minItems"
	if sMinItems := s.MinItems; sMinItems != 0 {
		schema["minItems"] = sMinItems
	}
	// "maxItems"
	if nil != s.MaxItems {
		schema["maxItems"] = *s.MaxItems
	}
	// "items"
	if sItems := s.Items; nil != sItems {
		schema["type"] = ensureSchemaType(schema["type"], "array")
		if sItems.Value.IsEmpty() {
			schema["items"] = []schemaJSON{}
		} else {
			schema["items"] = []schemaJSON{vald.schemaOrRefFromOA3(sItems)}
		}
	}

	// "minProperties"
	if sMinProps := s.MinProps; sMinProps != 0 {
		schema["minProperties"] = sMinProps
	}
	// "maxProperties"
	if nil != s.MaxProps {
		schema["maxProperties"] = *s.MaxProps
	}
	// "required"
	if sRequired := s.Required; len(sRequired) != 0 {
		schema["required"] = sRequired
	}
	// "properties"
	if count := len(s.Properties); count != 0 {
		schema["type"] = ensureSchemaType(schema["type"], "object")
		properties := make(schemasJSON, count)
		i, props := 0, make([]string, count)
		for propName := range s.Properties {
			props[i] = propName
			i++
		}
		sort.Strings(props)

		for j := 0; j != i; j++ {
			propName := props[j]
			properties[propName] = vald.schemaOrRefFromOA3(s.Properties[propName])
		}
		schema["properties"] = properties
	}
	//FIXME: "additionalProperties"
	if sAddProps := s.AdditionalPropertiesAllowed; sAddProps != nil {
		schema["additionalProperties"] = sAddProps
	}

	// "allOf"
	if sAllOf := s.AllOf; len(sAllOf) != 0 {
		allOf := make([]schemaJSON, len(sAllOf))
		for i, sOf := range sAllOf {
			allOf[i] = vald.schemaOrRefFromOA3(sOf)
		}
		schema["allOf"] = allOf
	}

	// "anyOf"
	if sAnyOf := s.AnyOf; len(sAnyOf) != 0 {
		anyOf := make([]schemaJSON, len(sAnyOf))
		for i, sOf := range sAnyOf {
			anyOf[i] = vald.schemaOrRefFromOA3(sOf)
		}
		schema["anyOf"] = anyOf
	}

	// "oneOf"
	if sOneOf := s.OneOf; len(sOneOf) != 0 {
		oneOf := make([]schemaJSON, len(sOneOf))
		for i, sOf := range sOneOf {
			oneOf[i] = vald.schemaOrRefFromOA3(sOf)
		}
		schema["oneOf"] = oneOf
	}

	// "not"
	if sNot := s.Not; nil != sNot {
		schema["not"] = vald.schemaOrRefFromOA3(sNot)
	}

	return
}

func ensureSchemaType(types interface{}, t string) []string {
	if types == nil {
		return []string{t}
	}
	ts := types.([]string)
	for _, aT := range ts {
		if t == aT {
			return ts
		}
	}
	return append(ts, t)
}

func pathFromOA3(inputs []*ParamJSON, basePath, path string) (partials []*PathPartial) {
	if basePath != "/" {
		p := &PathPartial{Pp: &PathPartial_Part{basePath}}
		partials = append(partials, p)
	}

	onCurly := func(r rune) bool { return r == '{' || r == '}' }
	isCurly := '{' == path[0]
	for i, part := range strings.FieldsFunc(path, onCurly) {
		var p PathPartial
		if isCurly || i%2 != 0 {
			ptr := sid(0)
			for _, param := range inputs {
				if part == param.Name {
					ptr = param.SID
				}
			}
			if ptr == 0 {
				panic(`can't find parameter for path param ` + part)
			}
			p.Pp = &PathPartial_Ptr{ptr}
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

	case "100" <= code && code <= "599":
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
