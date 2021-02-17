package starlarktruth

import (
	"fmt"
)

var _ error = (*IntegrityError)(nil)

// IntegrityError describes that an `assert.that(actual)` was called but never any of its `.truth_methods(subject)`.
// At the exception of `.named(name)` as by itself this still requires an assertion.
type IntegrityError string

func (e IntegrityError) Error() string {
	return fmt.Sprintf("%s: %s.that(...) is missing an assertion", string(e), Default)
}
