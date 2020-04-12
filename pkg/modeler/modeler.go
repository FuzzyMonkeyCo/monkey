package modeler

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

var (
	ErrUnparsablePayload = errors.New("unparsable piped payload")
	ErrNoSuchSchema      = errors.New("no such schema")
	ErrNoSuchRef         = errors.New("no such ref")
)

// Interface describes checkable models
type Interface interface {
	ToProto() *fm.Clt_Fuzz_Model
	FromProto(*fm.Clt_Fuzz_Model) error

	NewFromKwargs(starlark.StringDict) (Interface, *Error)

	SetResetter(resetter.Interface)
	GetResetter() resetter.Interface

	Lint(context.Context, bool) error

	InputsCount() int
	WriteAbsoluteReferences(io.Writer)
	FilterEndpoints([]string) ([]uint32, error)

	ValidateAgainstSchema(string, []byte) error
	Validate(uint32, *types.Value) []string

	NewCaller(context.Context, *fm.Srv_Call, func(string, ...interface{})) (Caller, error)

	// Check(...) ...
}

// Func TODO
type Func func(starlark.StringDict) (Interface, *Error)

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
