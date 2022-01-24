package runtime

import (
	"errors"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler/openapiv3"
	"go.starlark.net/starlark"
)

type builtin func(
	th *starlark.Thread,
	b *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (ret starlark.Value, err error)

type module struct {
	rt *Runtime

	bOpenAPIv3 builtin
}

var _ starlark.HasAttrs = (*module)(nil)

func (rt *Runtime) newModule() (m *module) {
	m = &module{
		rt: rt,
	}

	var mdlr modeler.Interface
	mdlr = (*openapiv3.T)(nil)
	log.Printf("[DBG] registering modeler: %q", "OpenAPIv3")
	m.bOpenAPIv3 = rt.modelMaker("OpenAPIv3", mdlr.NewFromKwargs)
	return
}

func (m *module) String() string        { return "monkey" }
func (m *module) Type() string          { return "monkey" }
func (m *module) Freeze()               {}
func (m *module) Truth() starlark.Bool  { return true }
func (m *module) Hash() (uint32, error) { return 0, errors.New("unhashable type: monkey") }

func (m *module) AttrNames() []string {
	return []string{
		"check",
		"env",
		"OpenAPIv3",
	}
}

func (m *module) Attr(name string) (starlark.Value, error) {
	switch name {
	case "check":
		b := starlark.NewBuiltin(name, m.rt.bCheck)
		return b.BindReceiver(m), nil
	case "env":
		b := starlark.NewBuiltin(name, m.rt.bEnv)
		return b.BindReceiver(m), nil
	case "OpenAPIv3":
		b := starlark.NewBuiltin(name, m.bOpenAPIv3)
		return b.BindReceiver(m), nil
	default:
		return nil, nil // no such method
	}
}
