package progresser

import (
	"context"
)

// Interface displays calls, resets and checks progression
type Interface interface {
	// WithContext sets ctx of a progresser.Interface implementation
	WithContext(context.Context)
	// MaxTestsCount sets an upper bound before testing starts
	MaxTestsCount(uint32)

	// TotalTestsCount may be called many times during testing
	TotalTestsCount(uint32)
	// TotalCallsCount may be called many times during testing
	TotalCallsCount(uint32)
	// TotalChecksCount may be called many times during testing
	TotalChecksCount(uint32)
	// TestCallsCount may be called many times during testing
	TestCallsCount(uint32)
	// CallChecksCount may be called many times during testing
	CallChecksCount(uint32)

	// CheckFailed may be called many times during testing
	CheckFailed(string, []string)
	// CheckSkipped may be called many times during testing
	CheckSkipped(string, string)
	// CheckPassed may be called many times during testing
	CheckPassed(string, string)
	// ChecksPassed may be called many times during testing
	ChecksPassed()

	// Printf formats informational data
	Printf(string, ...interface{})
	// Errorf formats error messages
	Errorf(string, ...interface{})

	// Before(Event, io.Writer)
	// After(Event, io.Writer)

	// Terminate cleans up after a progresser.Interface implementation instance
	Terminate() error
}
