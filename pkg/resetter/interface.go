package resetter

import (
	"context"
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

	// ExecStart executes the setup phase of the System Under Test
	ExecStart(context.Context, io.Writer, io.Writer, bool, map[string]string) error
	// ExecReset resets the System Under Test to a state similar to a post-ExecStart state
	ExecReset(context.Context, io.Writer, io.Writer, bool, map[string]string) error
	// ExecStop executes the cleanup phase of the System Under Test
	ExecStop(context.Context, io.Writer, io.Writer, bool, map[string]string) error

	// TidyOutput filter maps over each line
	TidyOutput([][]byte) TidiedOutput

	// Terminate cleans up after a resetter.Interface implementation instance
	Terminate(context.Context, io.Writer, io.Writer, map[string]string) error
}

type TidiedOutput [][]byte

var _ error = (*Error)(nil)

// Error describes a resetter error
type Error struct {
	bt TidiedOutput
}

// NewError returns a new empty resetter.Error
func NewError(bt TidiedOutput) *Error {
	return &Error{
		bt: bt,
	}
}

// Reason describes the error on multiple lines
func (re *Error) Reason() []string {
	bt := make([]string, 0, len(re.bt))
	for _, line := range re.bt {
		bt = append(bt, string(line))
	}
	return bt
}

// Error returns the error string
func (re *Error) Error() string {
	var msg strings.Builder
	msg.WriteByte('\n')
	msg.WriteString("script failed during Reset:")
	msg.WriteByte('\n')
	for _, line := range re.bt {
		msg.Write(line)
		msg.WriteByte('\n')
	}
	return msg.String()
}
