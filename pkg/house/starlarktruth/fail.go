package starlarktruth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

const warnContainsExactlySingleIterable = "" +
	" Passing a single iterable to .containsExactly(*expected) is often" +
	" not the correct thing to do. Did you mean to call" +
	" .containsExactlyElementsIn(Iterable) instead?"

func (t *T) unhandled(bName string, args ...starlark.Value) (starlark.Value, error) {
	// FIXME: make prettier
	err := fmt.Errorf("unhandled .%s%s", bName, starlark.Tuple(args).String())
	return nil, err
}

func errMustBeEqualNumberOfKVPairs(count int) error {
	return newInvalidAssertion(
		fmt.Sprintf("There must be an equal number of key/value pairs"+
			" (i.e., the number of key/value parameters (%d) must be even).", count))
}

func (t *T) failNone(check string, other starlark.Value) error {
	if other == starlark.None {
		msg := fmt.Sprintf("It is illegal to compare using .%s(None)", check)
		return newInvalidAssertion(msg)
	}
	return nil
}

func (t *T) failIterable() (starlark.Iterable, error) {
	itermap, ok := t.actual.(starlark.IterableMapping)
	if ok {
		iter := newTupleSlice(itermap.Items())
		return iter, nil
	}

	iter, ok := t.actual.(starlark.Iterable)
	if !ok {
		msg := fmt.Sprintf("Cannot use %s as Iterable.", t.subject())
		return nil, newInvalidAssertion(msg)
	}
	return iter, nil
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
	switch actual := t.actual.(type) {
	case starlark.String:
		if strings.Contains(actual.GoString(), "\n") {
			if t.name == "" {
				return "actual"
			}
			return fmt.Sprintf("actual %s", t.name)
		}
	case starlark.Tuple:
		if t.actualIsIterableFromString && len(actual) == 0 {
			// When printing an empty string that was turned into a tuple
			// it makes more sense to turn it back into a string
			// just to display it.
			t.actual = starlark.String("")
		}
	default:
	}
	if t.name == "" {
		return fmt.Sprintf("<%s>", t.actual.String())
	}
	return fmt.Sprintf("%s(<%s>)", t.name, t.actual.String())
}
