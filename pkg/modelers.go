package pkg

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"go.starlark.net/starlark"
)

var registeredIRModels = make(map[string]ModelerFunc)

// ModelerFunc TODO
type ModelerFunc func(d starlark.StringDict) (modeler.Modeler, *ModelerError)

// ModelerError TODO
type ModelerError struct {
	FieldRead, Want, Got string
}

func (me *ModelerError) Error(modelerName string) error {
	return fmt.Errorf("%s(%s = ...) must be %s, got: %s",
		modelerName, me.FieldRead, me.Want, me.Got)
}

// RegisterModeler TODO
func RegisterModeler(name string, fn ModelerFunc) {
	if _, ok := registeredIRModels[name]; ok {
		panic(fmt.Sprintf("modeler %q is already registered", name))
	}
	registeredIRModels[name] = fn
}
