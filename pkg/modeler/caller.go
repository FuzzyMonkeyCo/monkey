package modeler

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/gogo/protobuf/types"
)

// CheckerFunc returns whether validation succeeded, was skipped or failed.
type CheckerFunc func() (string, string, []string)

// Caller performs a request and awaits a response.
type Caller interface {
	ToProto() (*fm.Clt_CallRequestRaw, *fm.Clt_CallResponseRaw)

	Do(context.Context)

	// Request returns data one can use in their call checks.
	// It returns nil if the actual request could not be created.
	Request() *types.Struct
	// Response returns data one can use in their call checks.
	// It includes the req:=Request() and returns nil when req is.
	Response() *types.Struct

	NextCallerCheck() (string, CheckerFunc)
}
