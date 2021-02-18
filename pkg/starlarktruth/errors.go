package starlarktruth

import (
	"fmt"

	"go.starlark.net/starlark"
)

// InvalidAssertion signifies an invalid assertion was attempted
// such as comparing with None.
type InvalidAssertion string

var _ error = (InvalidAssertion)("")

func newInvalidAssertion(prop string) InvalidAssertion { return InvalidAssertion(prop) }
func (e InvalidAssertion) Error() string               { return string(e) }

// TruthAssertion signifies an assertion predicate was invalidated.
type TruthAssertion struct {
	e string
}

var _ error = (*TruthAssertion)(nil)

func newTruthAssertion(msg string) *TruthAssertion { return &TruthAssertion{e: msg} }
func (a *TruthAssertion) Error() string            { return a.e }

// unhandled internal & public errors

const errUnhandled = unhandledError(0)

type unhandledError int

var _ error = (unhandledError)(0)

func (e unhandledError) Error() string { return "unhandled" }

type UnhandledError struct {
	name   string
	actual starlark.Value
	args   starlark.Tuple
}

var _ error = (*UnhandledError)(nil)

func (t *T) unhandled(name string, args ...starlark.Value) *UnhandledError {
	return &UnhandledError{
		name:   name,
		actual: t.actual,
		args:   args,
	}
}
func (e UnhandledError) Error() string {
	return fmt.Sprintf("unhandled .%s with %s for %s", e.name, e.actual.String(), e.args.String())
}

var _ error = (*IntegrityError)(nil)

// IntegrityError describes that an `assert.that(actual)` was called but never any of its `.truth_methods(subject)`.
// At the exception of `.named(name)` as by itself this still requires an assertion.
type IntegrityError string

func (e IntegrityError) Error() string {
	return fmt.Sprintf("%s: %s.that(...) is missing an assertion", string(e), Default)
}
