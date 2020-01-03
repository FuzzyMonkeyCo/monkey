package modeler_openapiv3

import (
	"encoding/json"
	"log"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
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
			t.Logf("lint spec from OpenAPIv3 file")
			m0 := &oa3{}
			m0.File = docPath
			err := m0.Lint(false)
			require.NoError(t, err)
			t.Logf("validate some schemas")
			testSomeSchemas(t, m0)
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
			testSomeSchemas(t, m1)

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
			log.Printf("%s\n", blob1)
			m1.vald, err = newSpecFromOA3(&doc)
			require.NoError(t, err)
			testSomeSchemas(t, m1)

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

func toOA3(m *oa3) (doc openapi3.Swagger) {
	doc.OpenAPI = "3.0.0"
	doc.Info = openapi3.Info{
		Title:   someDescription,
		Version: "1.42.3",
	}
	var sm schemap
	sm = m.Spec.GetSchemas().GetJson()
	sm.schemasToOA3(&doc)
	sm.endpointsToOA3(&doc, m.Spec.GetEndpoints())
	return
}

func testSomeSchemas(t *testing.T, m *oa3) {
	require.NotEmpty(t, m.Validate(1, enumFromGo(schemaJSON{})))
	require.Empty(t, m.Validate(4, enumFromGo(float64(42))))
}
