package starlarktruth

import (
	"fmt"
)

// InvalidAssertion signifies an invalid assertion was attempted
// such as comparing with None.
type InvalidAssertion struct {
	p string
}

var _ error = (*InvalidAssertion)(nil)

func newInvalidAssertion(prop string) *InvalidAssertion { return &InvalidAssertion{p: prop} }
func (a *InvalidAssertion) Error() string {
	return fmt.Sprintf("It is illegal to compare using %s(None)", a.p)
}

// TruthAssertion signifies an assertion predicate was invalidated.
type TruthAssertion struct {
	e string
}

var _ error = (*TruthAssertion)(nil)

func newTruthAssertion(msg string) *TruthAssertion { return &TruthAssertion{e: msg} }
func (a *TruthAssertion) Error() string            { return a.e }
