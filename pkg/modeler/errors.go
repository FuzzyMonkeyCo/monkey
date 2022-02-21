package modeler

// NoSuchRefError represents a schema path that is not found
type NoSuchRefError struct {
	ref string
}

var _ error = (*NoSuchRefError)(nil)

func (e *NoSuchRefError) Error() string {
	return "no such ref: " + string(e.ref)
}

// NewNoSuchRefError creates a new error with the given absRef
func NewNoSuchRefError(ref string) *NoSuchRefError {
	return &NoSuchRefError{ref}
}
