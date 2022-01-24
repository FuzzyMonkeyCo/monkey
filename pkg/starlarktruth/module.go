package starlarktruth

import (
	"fmt"

	"go.starlark.net/starlark"
)

var (
	// Module is the module name used by default.
	Module = "assert"

	// Method is the attribute name used by default.
	Method = "that"
)

type module struct{}

var _ starlark.HasAttrs = (*module)(nil)

// NewModule registers a Starlark module of https://truth.dev/
func NewModule(predeclared starlark.StringDict) { predeclared[Module] = &module{} }

func (m *module) String() string        { return Module }
func (m *module) Type() string          { return Module }
func (m *module) Freeze()               {}
func (m *module) Truth() starlark.Bool  { return true }
func (m *module) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: %s", Module) }
func (m *module) AttrNames() []string   { return []string{Method} }
func (m *module) Attr(name string) (starlark.Value, error) {
	if name != Method {
		return nil, nil // no such method
	}
	b := starlark.NewBuiltin(name, That)
	return b.BindReceiver(m), nil
}

// That implements the `.that(target)` builtin
func That(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var target starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &target); err != nil {
		return nil, err
	}

	if err := Close(thread); err != nil {
		return nil, err
	}
	thread.SetLocal(LocalThreadKeyForClose, thread.CallFrame(1))

	return newT(target), nil
}

var _ starlark.HasAttrs = (*T)(nil)

func newT(target starlark.Value) *T { return &T{actual: target} }

func (t *T) String() string                           { return fmt.Sprintf("%s.%s(%s)", Module, Method, t.actual.String()) }
func (t *T) Type() string                             { return Module }
func (t *T) Freeze()                                  { t.actual.Freeze() }
func (t *T) Truth() starlark.Bool                     { return true }
func (t *T) Hash() (uint32, error)                    { return 0, fmt.Errorf("unhashable: %s", t.Type()) }
func (t *T) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name) }
func (t *T) AttrNames() []string                      { return attrNames }
