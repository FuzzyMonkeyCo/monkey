package modeler

import (
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"go.starlark.net/starlark"
)

// Modeler describes checkable models
type Modeler interface {
	ToProto() *fm.Clt_Msg_Fuzz_Model

	SetResetter(resetter.Resetter)
	GetResetter() resetter.Resetter

	Pretty(w io.Writer) (n int, err error)
}

// Func TODO
type Func func(d starlark.StringDict) (Modeler, *Error)

var _ error = (*Error)(nil)

// Error TODO
type Error struct {
	modelerName          string
	fieldRead, want, got string
}

func NewError(fieldRead, want, got string) *Error {
	return &Error{
		fieldRead: fieldRead,
		want:      want,
		got:       got,
	}
}

func (e *Error) SetModelerName(name string) {
	e.modelerName = name
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s(%s = ...) must be %s, got: %s",
		e.modelerName, e.fieldRead, e.want, e.got)
}
