package starlarktruth

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
