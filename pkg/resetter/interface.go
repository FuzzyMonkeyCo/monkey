package resetter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// Maker types the New func that instanciates new resetters
type Maker func(kwargs []starlark.Tuple) (Interface, error)

// Interface describes ways to reset the system under test to a known initial state
// A package defining a type that implements Interface also has to define:
// * a non-empty const Name that names the Starlark builtin
// * a func of type Maker named New that instanciates a new resetter
type Interface interface { // TODO: initers.Initer
	// Name uniquely identifies this instance
	Name() string

	// Provides lists the models a resetter resets
	Provides() []string

	// ToProto marshals a resetter.Interface implementation into a *fm.Clt_Fuzz_Resetter
	ToProto() *fm.Clt_Fuzz_Resetter

	// Env passes envs read during startup
	Env(read map[string]string)

	// ExecStart executes the setup phase of the System Under Test
	ExecStart(context.Context, io.Writer, io.Writer, bool) error
	// ExecReset resets the System Under Test to a state similar to a post-ExecStart state
	ExecReset(context.Context, io.Writer, io.Writer, bool) error
	// ExecStop executes the cleanup phase of the System Under Test
	ExecStop(context.Context, io.Writer, io.Writer, bool) error

	// Terminate cleans up after a resetter.Interface implementation instance
	Terminate(context.Context, io.Writer, io.Writer) error
}

var _ error = (*Error)(nil)

// Error describes a resetter error
type Error struct {
	bt []string
}

// NewError returns a new empty resetter.Error
func NewError(bt []string) *Error {
	return &Error{
		bt: bt,
	}
}

// Reason describes the error on multiple lines
func (re *Error) Reason() []string {
	return re.bt
}

// Error returns the error string
func (re *Error) Error() string {
	return fmt.Sprintf(
		"\nscript failed during Reset:\n%s",
		strings.Join(re.bt, "\n"),
	)
}
