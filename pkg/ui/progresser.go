package ui

import (
	"context"
)

// Progresser TODO
type Progresser interface {
	WithContext(context.Context)
	MaxTestsCount(uint32)

	TotalTestsCount(uint32)
	TotalCallsCount(uint32)
	TotalChecksCount(uint32)
	TestCallsCount(uint32)
	CallChecksCount(uint32)

	CheckFailed(string, []string)
	CheckSkipped(string, string)
	CheckPassed(string, string)
	ChecksPassed()

	Printf(string, ...interface{})
	Errorf(string, ...interface{})

	// Before(Event, io.Writer)
	// After(Event, io.Writer)

	Terminate() error
}
