package starlarktruth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

func (t *T) checkNone(proposition string, other starlark.Value) error {
	ok, err := starlark.EqualDepth(starlark.None, other, maxdepth)
	if err != nil {
		return err
	}
	if ok {
		return NewInvalidAssertion(proposition)
	}
	return nil
}

func (t *T) failComparingValues(verb string, other starlark.Value, suffix string) error {
	proposition := fmt.Sprintf("%s <%s>", verb, other.String())
	return t.failWithProposition(proposition, suffix)
}

func (t *T) failWithProposition(proposition, suffix string) error {
	msg := fmt.Sprintf("Not true that %s %s.%s", t.subject(), proposition, suffix)
	return t.fail(msg)
}

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
