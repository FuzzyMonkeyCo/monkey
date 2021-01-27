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
	// RequestProto returns call input as used by the client
	// and call output as received by the client.
	ToProto() (*fm.Clt_CallRequestRaw, *fm.Clt_CallResponseRaw)

	// Do sends the request and waits for the response
	Do(context.Context)

	// Request returns data one can use in their call checks.
	// It returns nil if the actual request could not be created.
	Request() *types.Struct
	// Response returns data one can use in their call checks.
	// It includes the req:=Request() and returns nil when req is.
	Response() *types.Struct

	// NextCallerCheck returns ("",nil) when out of checks to run.
	// Otherwise it returns named checks inherent to the caller.
	NextCallerCheck() (string, CheckerFunc)
}
