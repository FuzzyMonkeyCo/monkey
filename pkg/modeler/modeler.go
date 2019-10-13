package modeler

import (
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"go.starlark.net/starlark"
)

// Interface describes checkable models
type Interface interface {
	ToProto() *fm.Clt_Msg_Fuzz_Model

	NewFromKwargs(starlark.StringDict) (Interface, *Error)

	SetResetter(resetter.Interface)
	GetResetter() resetter.Interface

	NewCaller() Caller

	// Check(...) ...

	Pretty(w io.Writer) (n int, err error)
}

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
