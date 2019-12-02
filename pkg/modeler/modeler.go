package modeler

import (
	"errors"
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"go.starlark.net/starlark"
)

var (
	ErrUnparsablePayload = errors.New("unparsable piped payload")
	ErrNoSuchSchema      = errors.New("no such schema")
	ErrNoSuchRef         = errors.New("no such ref")
)

// Interface describes checkable models
type Interface interface {
	ToProto() *fm.Clt_Msg_Fuzz_Model
	FromProto(*fm.Clt_Msg_Fuzz_Model) error

	NewFromKwargs(starlark.StringDict) (Interface, *Error)

	SetResetter(resetter.Interface)
	GetResetter() resetter.Interface

	Lint(bool) error

	InputsCount() int
	WriteAbsoluteReferences(io.Writer)
	FilterEndpoints([]string) ([]uint32, error)

	ValidateAgainstSchema(string, []byte) error
	Validate(uint32, interface{}) []string

	NewCaller(*fm.Srv_Msg_Call, func(string, ...interface{})) (Caller, error)

	// Check(...) ...
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
