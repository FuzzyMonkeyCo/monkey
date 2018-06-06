package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLintOpenAPIv300Petstore(t *testing.T) {
	docPath, showSpec := "./misc/openapiv3.0.0_petstore.yaml", false
	blob, err := ioutil.ReadFile(docPath)
	spec, err := doLint(docPath, blob, showSpec)
	require.NoError(t, err)

	expected := &SpecIR{}
	require.Equal(t, expected, spec)
}

func TestLintOpenAPIv300PetstoreExpanded(t *testing.T) {
	docPath, showSpec := "./misc/openapiv3.0.0_petstore-expanded.yaml", false
	blob, err := ioutil.ReadFile(docPath)
	spec, err := doLint(docPath, blob, showSpec)
	require.NoError(t, err)

	expected := &SpecIR{}
	require.Equal(t, expected, spec)
}
