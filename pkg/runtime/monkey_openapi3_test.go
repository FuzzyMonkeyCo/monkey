package runtime

import (
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"github.com/stretchr/testify/require"
)

// generic over modelers

func TestModelMissingIsForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
print("Hullo")
`[1:])
	require.EqualError(t, err, `no models registered`)
	require.Nil(t, rt)
}

func TestModelPositionalArgsAreForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3("hi", name = "bla")
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3(...) does not take positional arguments, only named ones`[1:])
	require.Nil(t, rt)
}

func TestModelNamesMustBeLegal(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "blip blop",
    file = "some/api_spec.yml",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: only characters from `[1:]+tags.Alphabet+` should be in "blip blop"`)
	require.Nil(t, rt)
}

func TestModelNamesMustBeUnique(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "blip",
    file = "some/api_spec.yml",
)
monkey.openapi3(
    name = "blip",
    file = "some/api_spec.yml",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:5:16: in <toplevel>
Error in openapi3: a model named blip already exists`[1:])
	require.Nil(t, rt)

	rt, err = newFakeMonkey(t, `
monkey.openapi3(
    name = "blip",
    file = "some/api_spec.yml",
)
monkey.openapi3(
    name = "blop",
    file = "some/api_spec.yml",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:5:16: in <toplevel>
Error in openapi3: cannot define model blop as another (blip) already exists`[1:])
	require.Nil(t, rt)
}

// name

func TestOpenapi3NameIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    file = "some/api_spec.yml",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: missing argument for name`[1:])
	require.Nil(t, rt)
}

func TestOpenapi3NameTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: for parameter "name": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwargs

func TestOpenapi3AdditionalKwardsForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "mything",
    wef = "bla",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: unexpected keyword argument "wef"`[1:])
	require.Nil(t, rt)
}

// kwarg: file

func TestOpenapi3FileIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "some_name",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: missing argument for file`[1:])
	require.Nil(t, rt)
}

func TestOpenapi3FileTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "some_name",
    file = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: for parameter "file": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwarg: host

func TestOpenapi3HostTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "some_name",
    file = "some/api_spec.yml",
    host = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: for parameter "host": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwarg: header_authorization

func TestOpenapi3HeaderAuthorizationTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.openapi3(
    name = "some_name",
    file = "some/api_spec.yml",
    header_authorization = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:16: in <toplevel>
Error in openapi3: openapi3: for parameter "header_authorization": got float, want string`[1:])
	require.Nil(t, rt)
}
