package modeler

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// CheckerFunc returns whether validation succeeded, was skipped or failed.
type CheckerFunc func() (string, string, []string)

// Caller performs a request and awaits a response.
type Caller interface {
	// RequestProto returns call input as used by the client
	RequestProto() *fm.Clt_CallRequestRaw

	// Do sends the request and waits for the response
	Do(context.Context)

	// ResponseProto returns call output as received by the client
	ResponseProto() *fm.Clt_CallResponseRaw

	// NextCallerCheck returns ("",nil) when out of checks to run.
	// Otherwise it returns named checks inherent to the caller.
	NextCallerCheck() (string, CheckerFunc)
}
