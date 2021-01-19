package resetter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// Interface describes ways to reset the system under test to a known initial state
type Interface interface {
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
	Terminate(context.Context) error
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
