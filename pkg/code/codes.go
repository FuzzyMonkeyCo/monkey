package code

const (
	// OK is the success code
	OK = 0
	// Failed represents any non-specific failure
	Failed = 1
	// FailedLint means something happened during linting
	FailedLint = 2
	// FailedUpdate means `binName` executable could not be upgraded
	FailedUpdate = 3
	// FailedFmt means the formatting operation failed
	FailedFmt = 4
	// FailedFuzz means the fuzzing process found a bug with the system under test (SUT)
	FailedFuzz = 6
	// FailedExec means a user command (start, reset, stop) experienced failure
	FailedExec = 7
	// FailedSchema means the given payload does not validate provided schema
	FailedSchema = 9
)
