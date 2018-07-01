package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strconv"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

var xxx2uint32 = map[string]uint32{
	"default": 0,
	"1XX":     1,
	"2XX":     2,
	"3XX":     3,
	"4XX":     4,
	"5XX":     5,
}

func TestSpecXXX(t *testing.T) {
	for k, v := range xxx2uint32 {
		got, err := specXXX(k)
		require.NoError(t, err)
		require.Equal(t, v, got)
	}

	for i := 100; i < 600; i++ {
		k, v := strconv.Itoa(i), uint32(i)
		got, err := specXXX(k)
		require.NoError(t, err)
		require.Equal(t, v, got)
	}
}

func TestEncodeVersusEncodeDecodeEncode(t *testing.T) {
	log.Println("ignore me")
	for _, docPath := range []string{
		"./misc/openapiv3.0.0_petstore.yaml",
		"./misc/openapiv3.0.0_petstore.json",
		"./misc/openapiv3.0.0_petstore-expanded.yaml",
	} {
		t.Run(docPath, func(t *testing.T) {
			blob0, err := ioutil.ReadFile(docPath)
			require.NoError(t, err)
			spec0, err := doLint(docPath, blob0, false)
			require.NoError(t, err)
			require.NotNil(t, spec0)
			require.IsType(t, &SpecIR{}, spec0)
			bin0, err := proto.Marshal(spec0)
			require.NoError(t, err)
			require.NotNil(t, bin0)
			jsn0, err := new(jsonpb.Marshaler).MarshalToString(spec0)
			require.NoError(t, err)
			require.NotEmpty(t, jsn0)

			var spec1 SpecIR
			err = proto.Unmarshal(bin0, &spec1)
			require.NoError(t, err)
			require.NotNil(t, &spec1)
			doc := specToOpenAPIv3(&spec1)
			blob1, err := json.MarshalIndent(doc, "", "  ")
			require.NoError(t, err)
			t.Logf("blob1 = %s\n", blob1)
			spec2, err := doLint("bla.json", blob1, false)
			require.NoError(t, err)
			require.NotNil(t, spec2)
			require.IsType(t, &SpecIR{}, spec2)
			jsn1, err := new(jsonpb.Marshaler).MarshalToString(spec2)
			require.NoError(t, err)
			require.NotEmpty(t, jsn1)

			require.JSONEq(t, jsn0, jsn1)
			require.Equal(t, spec0, spec1)
			require.Equal(t, spec0, spec2)
		})
	}
}

func specToOpenAPIv3(spec *SpecIR) (doc openapi3.Swagger) {
	doc.OpenAPI = "3.0.0"
	doc.Info = openapi3.Info{
		Title:   "title",
		Version: "1.2.3",
	}

	specEndpoints := spec.Endpoints
	doc.Paths = make(openapi3.Paths, len(specEndpoints))
	for _, e := range specEndpoints {
		endpoint := e.GetJson()
		url := parameterizePath(endpoint.GetPath())
		path := &openapi3.PathItem{
			// Parameters:
		}

		op := &openapi3.Operation{
			// Parameters:
			// RequestBody:
			Responses: responses(endpoint.GetOutputs()),
		}
		setMethod(endpoint.GetMethod(), path, op)
		doc.Paths[url] = path
	}

	return
}

func responses(outs mapXXXToPtrOrSchema) openapi3.Responses {
	responses := make(openapi3.Responses, len(outs))
	for xxx, schema := range outs {
		XXX := xxx2XXX(xxx)
		responses[XXX] = &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: "placeholder",
				Content:     jsonContent(schema),
			},
		}
	}
	return responses
}

func jsonSchema(s *Schema_JSON) *openapi3.Schema {
	schema := openapi3.NewSchema()

	// enum
	//FIXME

	// type, nullable, types
	sTypes := s.GetType()
	if len(sTypes) == 1 {
		sType := sTypes[0]
		if sType == Schema_JSON_null {
			schema.Nullable = true
		} else {
			schema.Type = Schema_JSON_Type_name[int32(sType)]
		}
	} else {
		schema.Types = make([]string, len(sTypes))
		for i, t := range sTypes {
			schema.Types[i] = Schema_JSON_Type_name[int32(t)]
		}
	}

	// format
	schema.Format = s.GetFormat()

	// required, properties
	schema.Required = s.GetRequired()
	schema.Properties = make(map[string]*openapi3.SchemaRef)
	for propName, propSchema := range s.GetProperties() {
		var subSchema openapi3.SchemaRef
		if ptr := propSchema.GetPtr(); ptr != "" {
			subSchema.Ref = ptr
		} else {
			subSchema.Value = jsonSchema(propSchema.GetSchema())
		}
		schema.Properties[propName] = &subSchema
	}

	// allOf
	//FIXME

	return schema
}

func jsonContent(ps *PtrOrSchemaJSON) openapi3.Content {
	if ptr := ps.GetPtr(); ptr != "" {
		ref := openapi3.NewSchemaRef(ptr, nil)
		return openapi3.NewContentWithJSONSchemaRef(ref)
	}
	schema := jsonSchema(ps.GetSchema())
	return openapi3.NewContentWithJSONSchema(schema)
}

func xxx2XXX(xxx uint32) string {
	for k, v := range xxx2uint32 {
		if v == xxx {
			return k
		}
	}
	return strconv.FormatUint(uint64(xxx), 10)
}

func parameterizePath(path *Path) (s string) {
	for _, p := range path.GetPartial() {
		part := p.GetPart()
		if part != "" {
			s += part
		} else {
			s += "{" + p.GetPtr() + "}"
		}
	}
	return
}

func setMethod(m Method, p *openapi3.PathItem, op *openapi3.Operation) {
	switch m {
	//https://github.com/getkin/kin-openapi/pull/13
	// case Method_CONNECT : p.Connect = op
	case Method_DELETE:
		p.Delete = op
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
		p.Get = op
	}
}
