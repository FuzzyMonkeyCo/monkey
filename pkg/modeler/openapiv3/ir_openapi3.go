package openapiv3

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

const (
	mimeJSON             = "application/json"
	oa3ComponentsSchemas = "#/components/schemas/"
)

func newSpecFromOA3(doc *openapi3.T) (vald *validator, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	docPaths, docSchemas := doc.Paths, doc.Components.Schemas
	vald = newValidator(len(docPaths), len(docSchemas))
	log.Println("[DBG] seeding schemas")
	//TODO: use docPath as root of base
	if err = vald.schemasFromOA3(docSchemas); err != nil {
		return
	}

	var basePath string
	if basePath, err = doc.Servers.BasePath(); err != nil {
		log.Println("[ERR]", err)
		as.ColorERR.Println(err)
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
	return vald.seed(oa3ComponentsSchemas, schemas)
}

func (vald *validator) endpointsFromOA3(basePath string, docPaths openapi3.Paths) {
	paths := make([]string, 0, len(docPaths))
	for path := range docPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	i := 0
	for _, path := range paths {
		docOps := docPaths[path].Operations()
		methods := make([]string, 0, len(docOps))
		for docMethod := range docOps {
			methods = append(methods, docMethod)
		}
		sort.Strings(methods)

		for _, docMethod := range methods {
			i++
			log.Printf("[DBG] through #%d %s %s", i, docMethod, path)
			docOp := docOps[docMethod]
			var inputs []*fm.ParamJSON
			inputsCount := len(docOp.Parameters)
			if docOp.RequestBody != nil {
				inputsCount++
			}
			if inputsCount > 0 {
				inputs = make([]*fm.ParamJSON, 0, inputsCount)
				vald.inputsFromOA3(&inputs, docOp.Parameters)
				if docOp.RequestBody != nil {
					vald.inputBodyFromOA3(&inputs, docOp.RequestBody)
				}
			}
			outputs := vald.outputsFromOA3(docOp.Responses)
			method := methodFromOA3(docMethod)
			vald.Spec.Endpoints[eid(i)] = &fm.Endpoint{
				Endpoint: &fm.Endpoint_Json{
					Json: &fm.EndpointJSON{
						Method:       method,
						PathPartials: pathFromOA3(basePath, path),
						Inputs:       inputs,
						Outputs:      outputs,
					},
				},
			}
		}
	}
}

func (vald *validator) inputBodyFromOA3(inputs *[]*fm.ParamJSON, docReqBody *openapi3.RequestBodyRef) {
	//FIXME: handle .Ref
	docBody := docReqBody.Value
	for mime, ct := range docBody.Content {
		if mime == mimeJSON {
			docSchema := ct.Schema
			schema := vald.schemaOrRefFromOA3(docSchema)
			param := &fm.ParamJSON{
				IsRequired: docBody.Required,
				SID:        vald.ensureMapped(docSchema.Ref, schema),
				Name:       "",
				Kind:       fm.ParamJSON_body,
			}
			*inputs = append(*inputs, param)
			return
		}
	}
}

func (vald *validator) inputsFromOA3(inputs *[]*fm.ParamJSON, docParams openapi3.Parameters) {
	paramsCount := len(docParams)
	paramap := make(map[string]*openapi3.ParameterRef, paramsCount)
	names := make([]string, 0, paramsCount)
	for _, docParamRef := range docParams {
		docParam := docParamRef.Value
		name := docParam.In + docParam.Name
		names = append(names, name)
		paramap[name] = docParamRef
	}
	sort.Strings(names)

	for _, name := range names {
		docParamRef := paramap[name]
		//FIXME: handle .Ref
		docParam := docParamRef.Value
		kind := fm.ParamJSON_UNKNOWN
		switch docParam.In {
		case openapi3.ParameterInPath:
			kind = fm.ParamJSON_path
		case openapi3.ParameterInQuery:
			kind = fm.ParamJSON_query
		case openapi3.ParameterInHeader:
			kind = fm.ParamJSON_header
		case openapi3.ParameterInCookie:
			kind = fm.ParamJSON_cookie
		}
		docSchema := docParam.Schema
		schema := vald.schemaOrRefFromOA3(docSchema)
		param := &fm.ParamJSON{
			IsRequired: docParam.Required,
			SID:        vald.ensureMapped(docSchema.Ref, schema),
			Name:       docParam.Name,
			Kind:       kind,
		}
		*inputs = append(*inputs, param)
	}
}

func (vald *validator) outputsFromOA3(docResponses openapi3.Responses) (
	outputs map[uint32]sid,
) {
	outputs = make(map[uint32]sid)
	codes := make([]string, 0, len(docResponses))
	for code := range docResponses {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	for _, code := range codes {
		responseRef := docResponses[code]
		xxx := makeXXXFromOA3(code)
		// NOTE: Responses MAY have a schema
		if len(responseRef.Value.Content) == 0 {
			outputs[xxx] = 0
		}
		//FIXME: handle .Ref
		for mime, ct := range responseRef.Value.Content {
			if mime == mimeJSON {
				docSchema := ct.Schema
				if docSchema == nil {
					outputs[xxx] = 0
				} else {
					schema := vald.schemaOrRefFromOA3(docSchema)
					outputs[xxx] = vald.ensureMapped(docSchema.Ref, schema)
				}
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
		if sItems.Value != nil && sItems.Value.IsEmpty() {
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
		props := make([]string, 0, count)
		for propName := range s.Properties {
			props = append(props, propName)
		}
		sort.Strings(props)

		for _, propName := range props {
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
		allOf := make([]schemaJSON, 0, len(sAllOf))
		for _, sOf := range sAllOf {
			allOf = append(allOf, vald.schemaOrRefFromOA3(sOf))
		}
		schema["allOf"] = allOf
	}

	// "anyOf"
	if sAnyOf := s.AnyOf; len(sAnyOf) != 0 {
		anyOf := make([]schemaJSON, 0, len(sAnyOf))
		for _, sOf := range sAnyOf {
			anyOf = append(anyOf, vald.schemaOrRefFromOA3(sOf))
		}
		schema["anyOf"] = anyOf
	}

	// "oneOf"
	if sOneOf := s.OneOf; len(sOneOf) != 0 {
		oneOf := make([]schemaJSON, 0, len(sOneOf))
		for _, sOf := range sOneOf {
			oneOf = append(oneOf, vald.schemaOrRefFromOA3(sOf))
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

func pathFromOA3(basePath, path string) (partials []*fm.PathPartial) {
	if basePath != "/" {
		p := &fm.PathPartial{Pp: &fm.PathPartial_Part{Part: basePath}}
		partials = append(partials, p)
	}

	onCurly := func(r rune) bool { return r == '{' || r == '}' }
	isCurly := '{' == path[0]
	for i, part := range strings.FieldsFunc(path, onCurly) {
		var p fm.PathPartial
		if isCurly || i%2 != 0 {
			// TODO (vendor): ensure path params are part of inputs
			p.Pp = &fm.PathPartial_Ptr{Ptr: part}
		} else {
			p.Pp = &fm.PathPartial_Part{Part: part}
		}
		partials = append(partials, &p)
	}

	if length := len(partials); length > 1 {
		part1 := partials[0].GetPart()
		part2 := partials[1].GetPart()
		if part1 != "" && part2 != "" {
			partials = partials[1:]
			partials[0] = &fm.PathPartial{Pp: &fm.PathPartial_Part{Part: part1 + part2}}
			return
		}
	}
	return
}

func pathToOA3(partials []*fm.PathPartial) (s string) {
	for _, p := range partials {
		part := p.GetPart()
		if part != "" {
			s += part
		} else {
			s += "{" + p.GetPtr() + "}"
		}
	}
	return
}

var xxx2uint32 = map[string]uint32{
	"default": 0,
	"1XX":     1,
	"2XX":     2,
	"3XX":     3,
	"4XX":     4,
	"5XX":     5,
}

func fromStatusCode(code uint32) uint32 { return code / 100 }

func makeXXXFromOA3(code string) uint32 {
	if i, ok := xxx2uint32[code]; ok {
		return i
	}
	i, err := strconv.Atoi(code)
	if err != nil {
		panic(err)
	}
	return uint32(i)
}

func makeXXXToOA3(xxx uint32) string {
	for k, v := range xxx2uint32 {
		if v == xxx {
			return k
		}
	}
	return strconv.FormatUint(uint64(xxx), 10)
}

func isInputBody(input *fm.ParamJSON) bool {
	return input.GetName() == "" && input.GetKind() == fm.ParamJSON_body
}

func methodFromOA3(docMethod string) fm.EndpointJSON_Method {
	return fm.EndpointJSON_Method(fm.EndpointJSON_Method_value[docMethod])
}

func methodToOA3(m fm.EndpointJSON_Method, op *openapi3.Operation, p *openapi3.PathItem) {
	switch m {
	case fm.EndpointJSON_CONNECT:
		p.Connect = op
	case fm.EndpointJSON_DELETE:
		p.Delete = op
	case fm.EndpointJSON_GET:
		p.Get = op
	case fm.EndpointJSON_HEAD:
		p.Head = op
	case fm.EndpointJSON_OPTIONS:
		p.Options = op
	case fm.EndpointJSON_PATCH:
		p.Patch = op
	case fm.EndpointJSON_POST:
		p.Post = op
	case fm.EndpointJSON_PUT:
		p.Put = op
	case fm.EndpointJSON_TRACE:
		p.Trace = op
	default:
		panic(`no such method`)
	}
}
