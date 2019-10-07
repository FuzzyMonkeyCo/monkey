package code

const (
	OK     = 0
	Failed = 1
	// Something happened during linting
	FailedLint = 2
	// `binName` executable could not be upgraded
	FailedUpdate = 3
	// Some external dependency is missing (probably bash)
	FailedRequire = 5
	// Fuzzing found a bug!
	FailedFuzz = 6
	// A user command (start, reset, stop) failed
	FailedExec = 7
	// Validating payload against schema failed
	FailedSchema = 9
)
