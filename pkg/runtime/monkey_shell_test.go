package runtime

import (
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"github.com/stretchr/testify/require"
)

// generic over resetters

func TestResetterPositionalArgsAreForbidden(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell("hi", name="bla")
`)
	require.EqualError(t, err, `shell(...) does not take positional arguments, only named ones`)
	require.Nil(t, rt)
}

func TestResetterNamesMustBeLegal(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
	name = "blip blop",
	provides = ["some_model"],
)`)
	require.EqualError(t, err, `only characters from `+tags.Alphabet+` should be in "blip blop"`)
	require.Nil(t, rt)
}

func TestResetterNamesMustBeUnique(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
	name = "blip",
	provides = ["some_model"],
)
monkey.shell(
	name = "blip",
	provides = ["some_model"],
)`)
	require.EqualError(t, err, `a resetter named blip already exists`)
	require.Nil(t, rt)

	rt, err = newFakeMonkey(simplestPrelude + `
monkey.shell(
	name = "blip",
	provides = ["some_model"],
)
monkey.shell(
	name = "blop",
	provides = ["some_model"],
)`)
	require.NoError(t, err)
	require.Len(t, rt.resetters, 2)
}

// name

func TestShellNameIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
    reset = "true",
	provides = ["some_model"],
)`)
	require.EqualError(t, err, `shell: missing argument for name`)
	require.Nil(t, rt)
}

func TestShellNameTyping(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
    name = 42.1337,
	provides = ["some_model"],
)`)
	require.EqualError(t, err, `shell: for parameter "name": got float, want string`)
	require.Nil(t, rt)
}

// kwargs

func TestShellAdditionalKwardsForbidden(t *testing.T) {
	rt, err := newFakeMonkey(`
monkey.shell(
	name = "blop",
	provides = ["some_model"],
    wef = "bla",
)`[1:])
	require.EqualError(t, err, `shell: unexpected keyword argument "wef"`)
	require.Nil(t, rt)
}

// kwarg: provides

func TestShellProvidesIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
	name = "blop",
)`)
	require.EqualError(t, err, `shell: missing argument for provides`)
	require.Nil(t, rt)
}

func TestShellProvidesTyping(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
	name = "blop",
	provides = [42.1337],
)`)
	require.EqualError(t, err, `shell: for parameter "provides": got float, want string`)
	require.Nil(t, rt)
}

func TestShellProvidesNonEmpty(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
monkey.shell(
	name = "blop",
	provides = [],
)`)
	require.EqualError(t, err, `shell: for parameter "provides": must not be empty`)
	require.Nil(t, rt)
}

// kwarg: file

func TestShellStartTyping(t *testing.T) {
	rt, err := newFakeMonkey(`
monkey.shell(
	name = "blop",
	provides = ["some_model"],
    start = 42.1337,
)`[1:])
	require.EqualError(t, err, `shell: for parameter "start": got float, want string`)
	require.Nil(t, rt)
}

// kwarg: reset

func TestShellResetTyping(t *testing.T) {
	rt, err := newFakeMonkey(`
monkey.shell(
	name = "blop",
	provides = ["some_model"],
    reset = 42.1337,
)`[1:])
	require.EqualError(t, err, `shell: for parameter "reset": got float, want string`)
	require.Nil(t, rt)
}

// kwarg: stop

func TestShellStopTyping(t *testing.T) {
	rt, err := newFakeMonkey(`
monkey.shell(
	name = "blop",
	provides = ["some_model"],
    stop = 42.1337,
)`[1:])
	require.EqualError(t, err, `shell: for parameter "stop": got float, want string`)
	require.Nil(t, rt)
}
