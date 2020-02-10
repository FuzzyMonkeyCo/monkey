package starlarktruth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

func (t *T) failNone(proposition string, other starlark.Value) error {
	if other == starlark.None {
		return newInvalidAssertion(proposition)
	}
	return nil
}

func (t *T) failComparingValues(verb string, other starlark.Value, suffix string) error {
	proposition := fmt.Sprintf("%s <%s>", verb, other.String())
	return t.failWithProposition(proposition, suffix)
}

func (t *T) failWithProposition(proposition, suffix string) error {
	msg := fmt.Sprintf("Not true that %s %s.%s", t.subject(), proposition, suffix)
	return newTruthAssertion(msg)
}

func (t *T) failWithBadResults(
	verb string, other starlark.Value,
	fail_verb string, actual fmt.Stringer,
	suffix string,
) error {
	msg := fmt.Sprintf("%s <%s>. It %s <%s>",
		verb, other.String(),
		fail_verb, actual.String())
	return t.failWithProposition(msg, suffix)
}

// func (t *T) failWithSubject(verb string) error {
// 	msg := fmt.Sprintf("%s %s", t.subject(), verb)
// 	return t.fail(msg)
// }

func (t *T) subject() string {
	if s, ok := t.actual.(starlark.String); ok {
		if strings.Contains(s.GoString(), "\n") {
			if t.name == "" {
				return "actual"
			}
			return fmt.Sprintf("actual %s", t.name)
		}
	}
	if t.name == "" {
		return fmt.Sprintf("<%s>", t.actual.String())
	}
	return fmt.Sprintf("%s<%s>", t.name, t.actual.String())
}
