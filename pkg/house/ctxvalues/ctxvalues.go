// Values set by runtime to be accessible through context
// in other packages

package ctxvalues

// T is an integer type for context.Context keys
type T int32

const (
	// UserAgent key's associated value contains monkey version and platform information
	UserAgent T = iota
)
