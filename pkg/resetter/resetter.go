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
	ToProto() *fm.Clt_Fuzz_Resetter

	Env(read map[string]string)

	ExecStart(context.Context, io.Writer, io.Writer, bool) error
	ExecReset(context.Context, io.Writer, io.Writer, bool) error
	ExecStop(context.Context, io.Writer, io.Writer, bool) error

	Terminate(context.Context, bool) error
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
	return fmt.Sprintf("script failed during Reset:\n%s",
		strings.Join(re.bt, "\n"))
}
