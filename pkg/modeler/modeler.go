package modeler

import (
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/do/fuzz/reset"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"go.starlark.net/starlark"
)

var registeredIRModels = make(map[string]Func)

// Func TODO
type Func func(d starlark.StringDict) (Modeler, *Error)

// Modeler describes checkable models
type Modeler interface {
	ToProto() *fm.Clt_Msg_Fuzz_Model

	SetSUTResetter(reset.SUTResetter)
	GetSUTResetter() reset.SUTResetter

	Pretty(w io.Writer) (n int, err error)
}

// Error TODO
type Error struct {
	FieldRead, Want, Got string
}

func (me *Error) Error(modelerName string) error {
	return fmt.Errorf("%s(%s = ...) must be %s, got: %s",
		modelerName, me.FieldRead, me.Want, me.Got)
}

// RegisterModeler TODO
func RegisterModeler(name string, fn Func) {
	if _, ok := registeredIRModels[name]; ok {
		panic(fmt.Sprintf("modeler %q is already registered", name))
	}
	registeredIRModels[name] = fn
}
