package modeler

import (
	"context"
	"errors"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/gogo/protobuf/types"
)

var (
	ErrCallFailed  = errors.New("call to SUT unexpectedly failed")
	ErrCheckFailed = errors.New("call check failed")
)

// CheckerFunc TODO
type CheckerFunc func() (string, []string)

// Caller TODO
type Caller interface {
	ToProto() *fm.Clt_CallResponseRaw

	Do(context.Context) error

	Request() *types.Struct
	Response() *types.Struct

	// Check(...) ...
	// FIXME: really not sure that this belongs here:
	CheckFirst() (string, CheckerFunc)
}

// CaptureShower TODO
type CaptureShower interface {
	ShowRequest(func(string, ...interface{})) error
	ShowResponse(func(string, ...interface{})) error
}
