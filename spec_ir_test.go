package main

import (
	"strconv"
	"testing"

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
	//TODO
}
