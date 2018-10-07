package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

const someText = "some text"

//FIXME: use unrmarshalling to schemaref?
type schemap struct {
	M map[sid]*RefOrSchemaJSON
}

var xxx2uint32 = map[string]uint32{
	"default": 0,
	"1XX":     1,
	"2XX":     2,
	"3XX":     3,
	"4XX":     4,
	"5XX":     5,
}

func TestMakeXXXFromOA3(t *testing.T) {
	for k, v := range xxx2uint32 {
		got := makeXXXFromOA3(k)
		require.Equal(t, v, got)
	}

	for i := 100; i < 600; i++ {
		k, v := strconv.Itoa(i), uint32(i)
		got := makeXXXFromOA3(k)
		require.Equal(t, v, got)
	}
}

func TestEncodeVersusEncodeDecodeEncode(t *testing.T) {
	jsoner := &jsonpb.Marshaler{Indent: "\t"}
	for _, docPath := range []string{
		"./misc/openapiv3.0.0_petstore.yaml",
		"./misc/openapiv3.0.0_petstore.json",
		"./misc/openapiv3.0.0_petstore-expanded.yaml",
	} {
		t.Run(docPath, func(t *testing.T) {
			blob0, err := ioutil.ReadFile(docPath)
			require.NoError(t, err)
			vald0, err := doLint(docPath, blob0, false)
			require.NoError(t, err)
			require.NotNil(t, vald0.Spec)
			require.IsType(t, &SpecIR{}, vald0.Spec)
			bin0, err := proto.Marshal(vald0.Spec)
			require.NoError(t, err)
			require.NotNil(t, bin0)
			jsn0, err := jsoner.MarshalToString(vald0.Spec)
			require.NoError(t, err)
			require.NotEmpty(t, jsn0)

			var spec1 SpecIR
			err = proto.Unmarshal(bin0, &spec1)
			require.NoError(t, err)
			require.NotNil(t, &spec1)
			doc := specToOA3(&spec1)
			blob1, err := json.MarshalIndent(doc, "", "  ")
			// log.Printf("%s\n", blob1)
			require.NoError(t, err)
			log.Println("here we go again")
			vald2, err := doLint("bla.json", blob1, false)
			require.NoError(t, err)
			require.NotNil(t, vald2.Spec)
			require.IsType(t, &SpecIR{}, vald2.Spec)
			jsn1, err := jsoner.MarshalToString(vald2.Spec)
			require.NoError(t, err)
			require.NotEmpty(t, jsn1)

			require.JSONEq(t, jsn0, jsn1)
		})
	}
}

func specToOA3(spec *SpecIR) (doc openapi3.Swagger) {
	doc.OpenAPI = "3.0.0"
	doc.Info = openapi3.Info{
		Title:   someText,
		Version: "1.42.3",
	}
	sm := &schemap{M: spec.GetSchemas().GetJson()}
	sm.schemasToOA3(&doc)
	sm.endpointsToOA3(&doc, spec.GetEndpoints())
	return
}

func (sm *schemap) schemasToOA3(doc *openapi3.Swagger) {
	seededSchemas := make(map[string]*openapi3.SchemaRef, len(sm.M))
	for _, refOrSchema := range sm.M {
		if schemaPtr := refOrSchema.GetPtr(); schemaPtr != nil {
			if ref := schemaPtr.GetRef(); ref != "" {
				name := strings.TrimPrefix(ref, "#/components/schemas/")
				refd := sm.M[schemaPtr.GetSID()]
				seededSchemas[name] = sm.schemaToOA3(refd.GetSchema())
			}
		}
	}
	doc.Components.Schemas = seededSchemas
}

func (sm *schemap) endpointsToOA3(doc *openapi3.Swagger, es []*Endpoint) {
	doc.Paths = make(openapi3.Paths, len(es))
	for _, e := range es {
		endpoint := e.GetJson()
		url := pathToOA3(endpoint.GetPathPartials())
		inputs := endpoint.GetInputs()
		reqBody := sm.inputBodyToOA3(inputs)
		params := sm.inputsToOA3(inputs)
		op := &openapi3.Operation{
			RequestBody: reqBody,
			Parameters:  params,
			Responses:   sm.outputsToOA3(endpoint.GetOutputs()),
		}
		if doc.Paths[url] == nil {
			doc.Paths[url] = &openapi3.PathItem{}
		}
		methodToOA3(endpoint.GetMethod(), op, doc.Paths[url])
	}
}

func isInputBody(input *ParamJSON) bool {
	return input.GetName() == "" && input.GetKind() == ParamJSON_body
}

func (sm *schemap) inputBodyToOA3(inputs []*ParamJSON) (reqBodyRef *openapi3.RequestBodyRef) {
	if len(inputs) > 0 {
		body := inputs[0]
		if body != nil && isInputBody(body) {
			reqBody := &openapi3.RequestBody{
				Content:     sm.contentToOA3(body.GetSID()),
				Required:    body.GetRequired(),
				Description: someText,
			}
			reqBodyRef = &openapi3.RequestBodyRef{Value: reqBody}
		}
	}
	return
}

func (sm *schemap) inputsToOA3(inputs []*ParamJSON) (params openapi3.Parameters) {
	for _, input := range inputs {
		if isInputBody(input) {
			continue
		}

		var in string
		switch input.GetKind() {
		case ParamJSON_path:
			in = openapi3.ParameterInPath
		case ParamJSON_query:
			in = openapi3.ParameterInQuery
		case ParamJSON_header:
			in = openapi3.ParameterInHeader
		case ParamJSON_cookie:
			in = openapi3.ParameterInCookie
		}

		param := &openapi3.Parameter{
			Name:        input.GetName(),
			Required:    input.GetRequired(),
			In:          in,
			Description: someText,
			Schema:      sm.derefSchemaPtr(input.GetSID()),
		}

		params = append(params, &openapi3.ParameterRef{Value: param})
	}
	return
}

func (sm *schemap) outputsToOA3(outs map[uint32]sid) openapi3.Responses {
	responses := make(openapi3.Responses, len(outs))
	for xxx, schema := range outs {
		XXX := xxx2XXX(xxx)
		responses[XXX] = &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: someText,
				Content:     sm.contentToOA3(schema),
			},
		}
	}
	return responses
}

func (sm *schemap) contentToOA3(SID sid) openapi3.Content {
	schemaRef := sm.derefSchemaPtr(SID)
	return openapi3.NewContentWithJSONSchemaRef(schemaRef)
}

func (sm *schemap) derefSchemaPtr(SID sid) *openapi3.SchemaRef {
	s, ok := sm.M[SID]
	if !ok {
		panic(`schemaptr's SID must be in schemap`)
	}

	if ss := s.GetSchema(); ss != nil {
		if sp := s.GetPtr(); sp != nil {
			panic(`sub schemaptr must not be set`)
		}
		schema := sm.schemaToOA3(ss)
		schema.Ref = schemaPtr.GetRef()
		return schema
	}
	return sm.derefSchemaPtr(s.GetPtr())
}

func (sm *schemap) schemaToOA3(s *Schema_JSON) *openapi3.SchemaRef {
	schema := openapi3.NewSchema()

	// "enum"
	if sEnum := s.GetEnum(); len(sEnum) != 0 {
		schema.Enum = make([]interface{}, len(sEnum))
		for i, v := range sEnum {
			schema.Enum[i] = enumToOA3(v)
		}
	}

	// "type", "nullable"
	for _, t := range s.GetTypes() {
		if t == Schema_JSON_UNKNOWN {
			panic(`no way this is ever zero`)
		}
		if t == Schema_JSON_null {
			schema.Nullable = true
		} else {
			schema.Type = Schema_JSON_Type_name[int32(t)]
		}
	}

	// "format"
	schema.Format = formatToOA3(s.GetFormat())
	// "minLength"
	schema.MinLength = s.GetMinLength()
	// "maxLength"
	if s.GetHasMaxLength() {
		v := s.GetMaxLength()
		schema.MaxLength = &v
	}
	// "pattern"
	schema.Pattern = s.GetPattern()

	// "minimum"
	if s.GetHasMinimum() {
		v := s.GetMinimum()
		schema.Min = &v
	}
	// "maximum"
	if s.GetHasMaximum() {
		v := s.GetMaximum()
		schema.Max = &v
	}
	// "exclusiveMinimum", "exclusiveMaximum"
	schema.ExclusiveMin = s.GetExclusiveMinimum()
	schema.ExclusiveMax = s.GetExclusiveMaximum()
	// "multipleOf"
	if mulOf := s.GetTranslatedMultipleOf(); mulOf != 0.0 {
		v := mulOf + 1.0
		schema.MultipleOf = &v
	}

	// "uniqueItems"
	schema.UniqueItems = s.GetUniqueItems()
	// "minItems"
	schema.MinItems = s.GetMinItems()
	// "maxItems"
	if s.GetHasMaxItems() {
		v := s.GetMaxItems()
		schema.MaxItems = &v
	}
	// "items"
	if sItems := s.GetItems(); len(sItems) == 1 {
		schema.Items = sm.derefSchemaPtr(sItems[0])
	}

	// "minProperties"
	schema.MinProps = s.GetMinProperties()
	// "maxProperties"
	if s.GetHasMaxProperties() {
		v := s.GetMaxProperties()
		schema.MaxProps = &v
	}
	// "required"
	schema.Required = s.GetRequired()
	// "properties"
	if sProps := s.GetProperties(); len(sProps) != 0 {
		schema.Properties = make(map[string]*openapi3.SchemaRef, len(sProps))
		for propName, propSchema := range sProps {
			schema.Properties[propName] = sm.derefSchemaPtr(propSchema)
		}
	}

	// "allOf"
	if sAllOf := s.GetAllOf(); len(sAllOf) != 0 {
		schema.AllOf = make([]*openapi3.SchemaRef, len(sAllOf))
		for i, sOf := range sAllOf {
			schema.AllOf[i] = sm.derefSchemaPtr(sOf)
		}
	}

	// "AnyOf"
	if sAnyOf := s.GetAnyOf(); len(sAnyOf) != 0 {
		schema.AnyOf = make([]*openapi3.SchemaRef, len(sAnyOf))
		for i, sOf := range sAnyOf {
			schema.AnyOf[i] = sm.derefSchemaPtr(sOf)
		}
	}

	// "OneOf"
	if sOneOf := s.GetOneOf(); len(sOneOf) != 0 {
		schema.OneOf = make([]*openapi3.SchemaRef, len(sOneOf))
		for i, sOf := range sOneOf {
			schema.OneOf[i] = sm.derefSchemaPtr(sOf)
		}
	}

	// "Not"
	if sNot := s.GetNot(); 0 != sNot {
		schema.Not = sm.derefSchemaPtr(sNot)
	}

	return schema.NewRef()
}

func formatToOA3(format Schema_JSON_Format) string {
	switch format {
	case Schema_JSON_NONE:
		return ""
	case Schema_JSON_date_time:
		return "date-time"
	case Schema_JSON_uri_reference:
		return "uri-reference"
	default:
		return Schema_JSON_Format_name[int32(format)]
	}
}

func enumToOA3(value *ValueJSON) interface{} {
	if value.GetIsNull() {
		return nil
	}
	switch value.GetValue().(type) {
	case *ValueJSON_Boolean:
		return value.GetBoolean()
	case *ValueJSON_Number:
		return value.GetNumber()
	case *ValueJSON_Text:
		return value.GetText()
	case *ValueJSON_Array:
		val := value.GetArray().GetValues()
		vs := make([]interface{}, len(val))
		for i, v := range val {
			vs[i] = enumToOA3(v)
		}
		return vs
	case *ValueJSON_Object:
		val := value.GetObject().GetValues()
		vs := make(map[string]interface{}, len(val))
		for n, v := range val {
			vs[n] = enumToOA3(v)
		}
		return vs
	default:
		panic("unreachable")
	}
}

func xxx2XXX(xxx uint32) string {
	for k, v := range xxx2uint32 {
		if v == xxx {
			return k
		}
	}
	return strconv.FormatUint(uint64(xxx), 10)
}

func pathToOA3(partials []*PathPartial) (s string) {
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

func methodToOA3(m Method, op *openapi3.Operation, p *openapi3.PathItem) {
	switch m {
	case Method_CONNECT:
		p.Connect = op
	case Method_DELETE:
		p.Delete = op
	case Method_GET:
		p.Get = op
	case Method_HEAD:
		p.Head = op
	case Method_OPTIONS:
		p.Options = op
	case Method_PATCH:
		p.Patch = op
	case Method_POST:
		p.Post = op
	case Method_PUT:
		p.Put = op
	case Method_TRACE:
		p.Trace = op
	default:
		panic(`no such method`)
	}
}
