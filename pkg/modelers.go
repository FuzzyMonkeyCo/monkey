package pkg

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"go.starlark.net/starlark"
)

var registeredIRModels = make(map[string]ModelerFunc)

// RegisterModeler TODO
func RegisterModeler(name string, fn ModelerFunc) {
	if _, ok := registeredIRModels[name]; ok {
		panic(fmt.Sprintf("modeler %q is already registered", name))
	}
	registeredIRModels[name] = fn
}

// ModelerFunc TODO
type ModelerFunc func(d starlark.StringDict) (modeler.Modeler, *ModelerError)

var _ error = (*ModelerError)(nil)

// ModelerError TODO
type ModelerError struct {
	modelerName          string
	fieldRead, want, got string
}

func NewModelerError(fieldRead, want, got string) *ModelerError {
	return &ModelerError{
		fieldRead: fieldRead,
		want:      want,
		got:       got,
	}
}

func (me *ModelerError) SetModelerName(name string) {
	me.modelerName = name
}

func (me *ModelerError) Error() string {
	return fmt.Sprintf("%s(%s = ...) must be %s, got: %s",
		me.modelerName, me.fieldRead, me.want, me.got)
}
