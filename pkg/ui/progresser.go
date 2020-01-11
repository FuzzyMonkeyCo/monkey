package ui

// Progresser TODO
type Progresser interface {
	MaxTestsCount(uint32)

	TotalTestsCount(uint32)
	TotalCallsCount(uint32)
	TotalChecksCount(uint32)
	TestCallsCount(uint32)
	CallChecksCount(uint32)

	CheckFailed([]string)
	CheckSkipped(string)
	CheckPassed(string)
	ChecksPassed()

	Printf(string, ...interface{})
	Errorf(string, ...interface{})

	// Before(Event, io.Writer)
	// After(Event, io.Writer)

	Terminate() error
}
