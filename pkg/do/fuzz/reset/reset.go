package reset

import (
	"context"
	"fmt"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"go.starlark.net/starlark"
)

// SUTResetter describes ways to reset the system under test to a known initial state
type SUTResetter interface {
	ToProto() *fm.Clt_Msg_Fuzz_Resetter

	ExecStart(context.Context, fm.Client) error
	ExecReset(context.Context, fm.Client) error
	ExecStop(context.Context, fm.Client) error

	Terminate(context.Context, fm.Client) error
}

// NewFromKwargs TODO
func NewFromKwargs(modelerName string, r starlark.StringDict) (SUTResetter, error) {
	const (
		tExecReset = "ExecReset"
		tExecStart = "ExecStart"
		tExecStop  = "ExecStop"
	)
	var (
		ok bool
		v  starlark.Value
		vv starlark.String
		t  string
		// TODO: other SUTResetter.s
		resetter = &SUTShell{}
	)
	t = tExecStart
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.Start = vv.GoString()
	}
	t = tExecReset
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.Rst = vv.GoString()
	}
	t = tExecStop
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.Stop = vv.GoString()
	}
	if len(r) != 0 {
		return nil, fmt.Errorf("unexpected arguments to %s(): %s", modelerName, strings.Join(r.Keys(), ", "))
	}
	return resetter, nil
}


var _ error = (*Error)(nil)

// Error TODO
type Error struct {
	bt   []string
}

// NewError TODO
func NewError(bt []string) *Error {
	return &Error{
		bt:   bt,
	}
}

func (re *Error) reason() []string {
	return re.bt
}

// Error TODO
func (re *Error) Error() string {
	return fmt.Sprintf("script failed during Reset:\n%s",
		strings.Join(re.bt, "\n"))
}
