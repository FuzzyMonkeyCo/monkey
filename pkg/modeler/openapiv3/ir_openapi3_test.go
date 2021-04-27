package openapiv3

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/protovalue"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

var someDescription = "some description"

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
	pattern := filepath.Join("testdata", "specs", "openapi3", "*.*")
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.NotEmpty(t, matches)
	for _, docPath := range matches {
		t.Run(docPath, func(t *testing.T) {
			t.Logf("lint spec from OpenAPIv3 file")
			m0 := &oa3{}
			m0.File = docPath
			err := m0.Lint(context.TODO(), false)
			require.NoError(t, err)
			t.Logf("validate some schemas")
			validateSomeSchemas(t, m0)
			t.Logf("proto marshaling")
			proto0 := m0.ToProto()
			bin0, err := proto.Marshal(proto0)
			require.NoError(t, err)
			require.NotNil(t, bin0)
			jsn0, err := jsoner.MarshalToString(proto0)
			require.NoError(t, err)
			require.NotEmpty(t, jsn0)

			t.Logf("build & use validator from proto")
			var m1Prime fm.Clt_Fuzz_Model
			err = proto.Unmarshal(bin0, &m1Prime)
			require.NoError(t, err)
			require.NotNil(t, &m1Prime)
			m1 := &oa3{}
			err = m1.FromProto(&m1Prime)
			require.NoError(t, err)
			require.NotNil(t, m1.Clt_Fuzz_Model_OpenAPIv3)
			require.NotNil(t, m1.vald)
			validateSomeSchemas(t, m1)

			t.Logf("compare encoded to re-created")
			proto1 := m1.ToProto()
			jsn1, err := jsoner.MarshalToString(proto1)
			require.NoError(t, err)
			require.NotEmpty(t, jsn1)
			require.JSONEq(t, jsn0, jsn1)

			t.Logf("enc ∘ dec ∘ enc = enc")
			doc := toOA3(m1)
			blob1, err := json.MarshalIndent(doc, "", "  ")
			require.NoError(t, err)
			log.Printf("%s", append(blob1, '\n'))
			m1.vald, err = newSpecFromOA3(&doc)
			require.NoError(t, err)
			validateSomeSchemas(t, m1)

			t.Skipf("TODO: compare encoded with re-encoded")
			require.Equal(t, m1.vald.Spec, m0.vald.Spec)
			proto2 := m1.ToProto()
			jsn2, err := jsoner.MarshalToString(proto2)
			require.NoError(t, err)
			require.NotEmpty(t, jsn2)
			require.JSONEq(t, jsn0, jsn2)
		})
	}
}

func toOA3(m *oa3) (doc openapi3.T) {
	doc.OpenAPI = "3.0.0"
	doc.Info = &openapi3.Info{
		Title:   someDescription,
		Version: "1.42.3",
	}
	var sm schemap
	sm = m.Spec.GetSchemas().GetJson()
	sm.schemasToOA3(&doc)
	sm.endpointsToOA3(&doc, m.Spec.GetEndpoints())
	return
}

func validateSomeSchemas(t *testing.T, m *oa3) {
	require.NotEmpty(t, m.Validate(1, protovalue.FromGo(schemaJSON{})))
	require.Empty(t, m.Validate(4, protovalue.FromGo(float64(42))))
}

func (sm schemap) schemasToOA3(doc *openapi3.T) {
	seededSchemas := make(map[string]*openapi3.SchemaRef, len(sm))
	for _, refOrSchema := range sm {
		if schemaPtr := refOrSchema.GetPtr(); schemaPtr != nil {
			if ref := schemaPtr.GetRef(); ref != "" {
				name := strings.TrimPrefix(ref, oa3ComponentsSchemas)
				seededSchemas[name] = sm.schemaToOA3(schemaPtr.GetSID())
			}
		}
	}
	doc.Components.Schemas = seededSchemas
}

func (sm schemap) endpointsToOA3(doc *openapi3.T, es map[eid]*fm.Endpoint) {
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

func (sm schemap) inputBodyToOA3(inputs []*fm.ParamJSON) (reqBodyRef *openapi3.RequestBodyRef) {
	if len(inputs) > 0 {
		body := inputs[0]
		if body != nil && isInputBody(body) {
			reqBody := &openapi3.RequestBody{
				Content:     sm.contentToOA3(body.GetSID()),
				Required:    body.GetIsRequired(),
				Description: someDescription,
			}
			reqBodyRef = &openapi3.RequestBodyRef{Value: reqBody}
		}
	}
	return
}

func (sm schemap) inputsToOA3(inputs []*fm.ParamJSON) (params openapi3.Parameters) {
	for _, input := range inputs {
		if isInputBody(input) {
			continue
		}

		var in string
		switch input.GetKind() {
		case fm.ParamJSON_path:
			in = openapi3.ParameterInPath
		case fm.ParamJSON_query:
			in = openapi3.ParameterInQuery
		case fm.ParamJSON_header:
			in = openapi3.ParameterInHeader
		case fm.ParamJSON_cookie:
			in = openapi3.ParameterInCookie
		}

		param := &openapi3.Parameter{
			Name:        input.GetName(),
			Required:    input.GetIsRequired(),
			In:          in,
			Description: someDescription,
			Schema:      sm.schemaToOA3(input.GetSID()),
		}

		params = append(params, &openapi3.ParameterRef{Value: param})
	}
	return
}

func (sm schemap) outputsToOA3(outs map[uint32]sid) openapi3.Responses {
	responses := make(openapi3.Responses, len(outs))
	for xxx, SID := range outs {
		XXX := makeXXXToOA3(xxx)
		responses[XXX] = &openapi3.ResponseRef{
			Value: &openapi3.Response{Description: &someDescription}}
		if SID != 0 {
			responses[XXX].Value.Content = sm.contentToOA3(SID)
		}
	}
	return responses
}

func (sm schemap) contentToOA3(SID sid) openapi3.Content {
	schemaRef := sm.schemaToOA3(SID)
	return openapi3.NewContentWithJSONSchemaRef(schemaRef)
}

func (sm schemap) schemaToOA3(SID sid) *openapi3.SchemaRef {
	s := sm.toGo(SID)
	s = transformSchemaToOA3(s)

	sJSON, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	schema := openapi3.NewSchema()
	if err := json.Unmarshal(sJSON, &schema); err != nil {
		panic(err)
	}

	return schema.NewRef()
}

func transformSchemaToOA3(s schemaJSON) schemaJSON {
	// "type", "nullable"
	if v, ok := s["type"]; ok {
		sTypes := v.([]string)
		sType := ""
		for _, v := range sTypes {
			switch v {
			case "":
				continue
			case fm.Schema_JSON_null.String():
				s["nullable"] = true
			default:
				sType = v
			}
		}
		s["type"] = sType
	}

	// "items"
	if v, ok := s["items"]; ok {
		if vv := v.([]schemaJSON); len(vv) > 0 {
			s["items"] = transformSchemaToOA3(vv[0])
		}
	}

	// "properties"
	if v, ok := s["properties"]; ok {
		props := v.(schemaJSON)
		for propName, propSchema := range props {
			props[propName] = transformSchemaToOA3(propSchema.(schemaJSON))
		}
		s["properties"] = props
	}

	// "allOf"
	if v, ok := s["allOf"]; ok {
		allOf := v.([]schemaJSON)
		for i, schemaOf := range allOf {
			allOf[i] = transformSchemaToOA3(schemaOf)
		}
		s["allOf"] = allOf
	}

	// "anyOf"
	if v, ok := s["anyOf"]; ok {
		anyOf := v.([]schemaJSON)
		for i, schemaOf := range anyOf {
			anyOf[i] = transformSchemaToOA3(schemaOf)
		}
		s["anyOf"] = anyOf
	}

	// "oneOf"
	if v, ok := s["oneOf"]; ok {
		oneOf := v.([]schemaJSON)
		for i, schemaOf := range oneOf {
			oneOf[i] = transformSchemaToOA3(schemaOf)
		}
		s["oneOf"] = oneOf
	}

	// "not"
	if v, ok := s["not"]; ok {
		s["not"] = transformSchemaToOA3(v.(schemaJSON))
	}

	return s
}
