package starlarktruth

import (
	"fmt"

	"go.starlark.net/starlark"
)

// Default is the default module constructor. It is merely the string "assert".
const Default = "assert"

type module struct{}

var _ starlark.Value = (*module)(nil)
var _ starlark.HasAttrs = (*module)(nil)

// NewModule registers a Starlark module of https://truth.dev/
func NewModule(predeclared starlark.StringDict) { predeclared[Default] = &module{} }

func (m *module) String() string        { return Default }
func (m *module) Type() string          { return Default }
func (m *module) Freeze()               {}
func (m *module) Truth() starlark.Bool  { return false }
func (m *module) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: %s", Default) }
func (m *module) AttrNames() []string   { return []string{"that"} }
func (m *module) Attr(name string) (starlark.Value, error) {
	if name != "that" {
		return nil, nil // no such method
	}
	b := starlark.NewBuiltin(Default, that)
	return b.BindReceiver(m), nil
}

// Implements the `.that(target)` builtin
func that(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	//TODO? b.Receiver() to check closedness
	var target starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &target); err != nil {
		return nil, err
	}
	return newT(target), nil
}

var _ starlark.Value = (*T)(nil)
var _ starlark.HasAttrs = (*T)(nil)

func newT(target starlark.Value) *T { return &T{actual: target} }

func (t *T) String() string                           { return fmt.Sprintf("%s.that(_)", Default) }
func (t *T) Type() string                             { return Default }
func (t *T) Freeze()                                  { t.actual.Freeze() }
func (t *T) Truth() starlark.Bool                     { return t.actual.Truth() }
func (t *T) Hash() (uint32, error)                    { return t.actual.Hash() }
func (t *T) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name) }
func (t *T) AttrNames() []string                      { return attrNames }
