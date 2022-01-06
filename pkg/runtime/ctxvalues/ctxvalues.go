// Values set by monkey's runtime to be accessible through
// context in other packages

package ctxvalues

type xUserAgent struct{}

// XUserAgent key's associated value contains monkey version and platform information
var XUserAgent = xUserAgent{}
