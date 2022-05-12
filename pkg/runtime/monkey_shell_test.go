package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
)

// generic over resetters

func TestResetterPositionalArgsAreForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell("hi", name = "bla")
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell(...) does not take positional arguments, only named ones`[1:])
	require.Nil(t, rt)
}

func TestResetterNamesMustBeLegal(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blip blop",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: only characters from `[1:]+tags.Alphabet+` should be in "blip blop"`)
	require.Nil(t, rt)
}

func TestResetterNamesMustBeUnique(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blip",
    provides = ["some_model"],
)
monkey.shell(
    name = "blip",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:5:13: in <toplevel>
Error in shell: a resetter named blip already exists`[1:])
	require.Nil(t, rt)

	rt, err = newFakeMonkey(t, `
monkey.shell(
    name = "blip",
    provides = ["some_model"],
)
monkey.shell(
    name = "blop",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.resetters, 2)
}

// name

func TestShellNameIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    reset = "true",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: missing argument for name`[1:])
	require.Nil(t, rt)
}

func TestShellNameTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = 42.1337,
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "name": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwargs

func TestShellAdditionalKwardsForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    wef = "bla",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: unexpected keyword argument "wef"`[1:])
	require.Nil(t, rt)
}

// kwarg: provides

func TestShellProvidesIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: missing argument for provides`[1:])
	require.Nil(t, rt)
}

func TestShellProvidesTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = [42.1337],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "provides": got float, want string`[1:])
	require.Nil(t, rt)
}

func TestShellProvidesNonEmpty(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = [],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "provides": must not be empty`[1:])
	require.Nil(t, rt)
}

// kwarg: file

func TestShellStartTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    start = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "start": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwarg: reset

func TestShellResetTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    reset = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "reset": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwarg: stop

func TestShellStopTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    stop = 42.1337,
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "stop": got float, want string`[1:])
	require.Nil(t, rt)
}
