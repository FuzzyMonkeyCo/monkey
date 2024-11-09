package runtime

import (
	"errors"
	"fmt"
	"log"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	openapi3 "github.com/FuzzyMonkeyCo/monkey/pkg/modeler/openapiv3"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter/shell"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
)

const (
	moduleBuiltins  = 2
	moduleModelers  = 1
	moduleResetters = 1

	moduleAttrs = moduleBuiltins + moduleModelers + moduleResetters
)

type module struct {
	attrs map[string]*starlark.Builtin
}

var _ starlark.HasAttrs = (*module)(nil)

func (rt *Runtime) newModule() (m *module) {
	m = &module{
		attrs: make(map[string]*starlark.Builtin, moduleAttrs),
	}

	modelMaker := func(modelerName string, maker modeler.Maker) *starlark.Builtin {
		f := rt.modelMakerBuiltin(modelerName, maker)
		b := starlark.NewBuiltin(modelerName, f)
		return b.BindReceiver(m)
	}
	m.attrs["openapi3"] = modelMaker(openapi3.Name, openapi3.New)

	resetterMaker := func(resetterName string, maker resetter.Maker) *starlark.Builtin {
		f := rt.resetterMakerBuiltin(resetterName, maker)
		b := starlark.NewBuiltin(resetterName, f)
		return b.BindReceiver(m)
	}
	m.attrs["shell"] = resetterMaker(shell.Name, shell.New)

	m.attrs["check"] = starlark.NewBuiltin("check", rt.bCheck).BindReceiver(m)
	m.attrs["env"] = starlark.NewBuiltin("env", rt.bEnv).BindReceiver(m)
	return
}

func (m *module) AttrNames() []string {
	return []string{
		"check",
		"env",
		"openapi3",
		"shell",
	}
}

func (m *module) Attr(name string) (starlark.Value, error) {
	if v := m.attrs[name]; v != nil {
		return v, nil
	}
	return nil, nil // no such method
}

func (m *module) String() string        { return "monkey" }
func (m *module) Type() string          { return "monkey" }
func (m *module) Freeze()               {}
func (m *module) Truth() starlark.Bool  { return true }
func (m *module) Hash() (uint32, error) { return 0, errors.New("unhashable type: monkey") }

type builtin func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (ret starlark.Value, err error)

func (rt *Runtime) modelMakerBuiltin(modelerName string, maker modeler.Maker) builtin {
	return func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (ret starlark.Value, err error) {
		log.Printf("[DBG] registering new %s model", modelerName)
		ret = starlark.None

		if len(args) != 0 {
			err = fmt.Errorf("%s(...) does not take positional arguments, only named ones", b.Name())
			log.Println("[ERR]", err)
			return
		}

		var model modeler.Interface
		if model, err = maker(kwargs); err != nil {
			log.Println("[ERR]", b.Name(), err)
			return
		}

		modelName := model.Name()

		if err = tags.LegalName(modelName); err != nil {
			log.Println("[ERR]", b.Name(), err)
			return
		}

		if _, ok := rt.models[modelName]; ok {
			err = fmt.Errorf("a model named %s already exists", modelName)
			log.Println("[ERR]", err)
			return
		}
		rt.models[modelName] = model
		log.Printf("[NFO] registered %s: %q", b.Name(), modelName)
		if len(rt.modelsNames) == 1 { // TODO: support >1 models
			err = fmt.Errorf("cannot define model %s as another (%s) already exists", modelName, rt.modelsNames[0])
			log.Println("[ERR]", err)
			return
		}
		rt.modelsNames = append(rt.modelsNames, modelName)
		return
	}
}

func (rt *Runtime) resetterMakerBuiltin(resetterName string, maker resetter.Maker) builtin {
	return func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (ret starlark.Value, err error) {
		log.Printf("[DBG] registering new %s resetter", resetterName)
		ret = starlark.None

		if len(args) != 0 {
			err = fmt.Errorf("%s(...) does not take positional arguments, only named ones", b.Name())
			log.Println("[ERR]", err)
			return
		}

		var rsttr resetter.Interface
		if rsttr, err = maker(kwargs); err != nil {
			log.Println("[ERR]", b.Name(), err)
			return
		}

		rsttrName := rsttr.Name()

		if err = tags.LegalName(rsttrName); err != nil {
			log.Println("[ERR]", b.Name(), err)
			return
		}

		if _, ok := rt.resetters[rsttrName]; ok {
			err = fmt.Errorf("a resetter named %s already exists", rsttrName)
			log.Println("[ERR]", err)
			return
		}
		rt.resetters[rsttrName] = rsttr
		log.Printf("[NFO] registered %s: %q", b.Name(), rsttrName)
		rt.resettersNames = append(rt.resettersNames, rsttrName)
		return
	}
}
