package modeler

import (
	"context"
	"errors"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/gogo/protobuf/types"
)

var ErrCheckFailed = errors.New("call check failed")

// CheckerFunc returns whether validation succeeded, was skipped or failed.
type CheckerFunc func() (string, string, []string)

// Caller performs a request and awaits a response.
type Caller interface {
	ToProto() *fm.Clt_CallResponseRaw

	Do(context.Context)

	Request() *types.Struct
	Response() *types.Struct

	NextCallerCheck() (string, CheckerFunc)
}
