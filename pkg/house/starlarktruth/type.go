package starlarktruth

import (
	"go.starlark.net/starlark"
)

type T struct {
	// Target in AssertThat(target)
	actual starlark.Value

	// Readable optional prefix with .Named(name)
	name string

	// True when actual was a String and was made into an iterable.
	// Helps when pretty printing.
	actualIsIterableFromString bool

	// FIXME: closedness
	// Whether an AssertThat(x)... call chain was properly terminated
	closed bool

	// True when asserting order
	askedInOrder bool
}

func (t *T) turnActualIntoIterableFromString() {
	s := t.actual.(starlark.String).GoString()
	vs := make([]starlark.Value, 0, len(s))
	for _, c := range s {
		vs = append(vs, starlark.String(c))
	}
	t.actual = starlark.Tuple(vs)
	t.actualIsIterableFromString = true
}
