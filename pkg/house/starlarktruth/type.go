package starlarktruth

import (
	"go.starlark.net/starlark"
)

type T struct {
	// Target in AssertThat(target)
	actual starlark.Value
	// Readable optional prefix with .Named(name)
	name string
	// Whether an AssertThat(x)... call chain was properly terminated
	closed bool
}
