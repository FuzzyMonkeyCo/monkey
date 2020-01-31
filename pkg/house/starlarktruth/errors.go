package starlarktruth

// InvalidAssertion signifies an invalid assertion was attempted
// such as comparing with None.
type InvalidAssertion struct {
	e string
}

var _ error = (*InvalidAssertion)(nil)

func (a *InvalidAssertion) Error() string { return a.e }

// TruthAssertion signifies an assertion predicate was invalidated.
type TruthAssertion struct {
	e string
}

var _ error = (*TruthAssertion)(nil)

func (a *TruthAssertion) Error() string { return a.e }

func (t *T) fail(msg string) error {
	return &TruthAssertion{e: msg}
}
