package resetter

import (
	"context"
	"fmt"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// Interface describes ways to reset the system under test to a known initial state
type Interface interface {
	ToProto() *fm.Clt_Msg_Fuzz_Resetter

	ExecStart(context.Context, fm.Client) error
	ExecReset(context.Context, fm.Client) error
	ExecStop(context.Context, fm.Client) error

	Terminate(context.Context, fm.Client) error
}

var _ error = (*Error)(nil)

// Error TODO
type Error struct {
	bt []string
}

// NewError TODO
func NewError(bt []string) *Error {
	return &Error{
		bt: bt,
	}
}

func (re *Error) Reason() []string {
	return re.bt
}

// Error TODO
func (re *Error) Error() string {
	return fmt.Sprintf("script failed during Reset:\n%s",
		strings.Join(re.bt, "\n"))
}
