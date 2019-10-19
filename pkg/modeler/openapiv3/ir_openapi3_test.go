package modeler_openapiv3

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

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
			blob0, err := ioutil.ReadFile(docPath)
			require.NoError(t, err)
			vald0, err := DoLint(docPath, blob0, false)
			require.NoError(t, err)
			require.NotNil(t, vald0.Spec)
			require.IsType(t, &SpecIR{}, vald0.Spec)
			testSomeSchemas(t, vald0.Spec)
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
			vald2, err := DoLint("bla.json", blob1, false)
			require.NoError(t, err)
			require.NotNil(t, vald2.Spec)
			require.IsType(t, &SpecIR{}, vald2.Spec)
			testSomeSchemas(t, vald2.Spec)
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
		Title:   someDescription,
		Version: "1.42.3",
	}
	var sm schemap
	sm = spec.GetSchemas().GetJson()
	sm.schemasToOA3(&doc)
	sm.endpointsToOA3(&doc, spec.GetEndpoints())
	return
}

func testSomeSchemas(t *testing.T, spec *SpecIR) {
	ss := spec.Schemas
	require.NotEmpty(t, ss.Validate(1, schemaJSON{}))
	require.Empty(t, ss.Validate(4, 42))
}
