package main

import (
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestSpecXXX(t *testing.T) {
	for k, v := range map[string]uint32{
		"default": 0,
		"1XX":     1,
		"2XX":     2,
		"3XX":     3,
		"4XX":     4,
		"5XX":     5,
	} {
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
	for _, docPath := range []string{
		"./misc/openapiv3.0.0_petstore.yaml",
		"./misc/openapiv3.0.0_petstore.json",
		"./misc/openapiv3.0.0_petstore-expanded.yaml",
	} {
		t.Run(docPath, func(t *testing.T) {
			blob, err := ioutil.ReadFile(docPath)
			require.NoError(t, err)
			spec0, err := doLint(docPath, blob, false)
			require.NoError(t, err)
			require.NotNil(t, spec0)
			require.IsType(t, &SpecIR{}, spec0)
			bin0, err := proto.Marshal(spec0)
			require.NoError(t, err)
			require.NotNil(t, bin0)

			var spec1 SpecIR
			err = proto.Unmarshal(bin0, &spec1)
			require.NoError(t, err)
			require.NotNil(t, &spec1)
			//TODO: specToOpenAPIv3(spec) then lint again then compare
		})
	}
}
