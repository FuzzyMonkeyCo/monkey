package starlarktruth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Maximum nesting browsed when comparing values
const maxdepth = 10

// const (
// 	tyNoneType = "NoneType"
// 	tyBool="bool"
// 	tyInt="int"
// 	tyFloat="float"
// 	tyString="string"
// 	tyList ="list" // ptr
// 	tyTuple="tuple"
// 	tyDict="dict" // ptr
// 	tySet="set" // ptr
// 	// *Function
// 	// *Builtin
// )

func isNotEqualTo(t *T, args ...starlark.Value) (starlark.Value, error) {
	other := args[0]
	ok, err := starlark.CompareDepth(syntax.EQL, t.actual, other, maxdepth)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, t.failComparingValues("is not equal to", other, "")
	}
	return starlark.None, nil
}

func isEqualTo(t *T, args ...starlark.Value) (starlark.Value, error) {
	other := args[0]
	switch other := other.(type) {
	case starlark.String:
		if actual, ok := t.actual.(starlark.String); ok {
			a := actual.GoString()
			o := other.GoString()
			// Use unified diff strategy when comparing multiline strings.
			if strings.Contains(a, "\n") && strings.Contains(o, "\n") {
				diff := difflib.ContextDiff{
					A:        difflib.SplitLines(o),
					B:        difflib.SplitLines(a),
					FromFile: "Expected",
					ToFile:   "Actual",
					Context:  3,
					Eol:      "\n",
				}
				pretty, err := difflib.GetContextDiffString(diff)
				if err != nil {
					return nil, err
				}
				pretty = strings.Replace(pretty, "\t", " ", -1)
				msg := "is equal to expected, found diff:\n" + pretty
				return nil, t.failWithProposition(msg, "")
			}
		}
	case starlark.Indexable: // e.g. tuple, list
		return containsExactlyElementsInOrderIn(t, other)
	case starlark.Iterable: // e.g. dict, set
		return containsExactlyElementsIn(t, other)
	default:
	}
	ok, err := starlark.CompareDepth(syntax.NEQ, t.actual, other, maxdepth)
	if err != nil {
		return nil, err
	}
	if ok {
		suffix := ""
		if t.actual.String() == other.String() {
			suffix = " However, their str() representations are equal."
		}
		return nil, t.failComparingValues("is equal to", other, suffix)
	}
	return starlark.None, nil
}

func containsExactlyElementsInOrderIn(t *T, expected starlark.Value) (starlark.Value, error) {
	return t.containsExactlyElementsIn(expected, inOrder())
}

func containsExactlyElementsIn(t *T, expected starlark.Value) (starlark.Value, error) {
	return t.containsExactlyElementsIn(expected)
}

type containsOptions struct {
	inOrder        bool
	warnElementsIn bool
}

type containsOption func(*containsOptions)

func inOrder() containsOption        { return func(o *containsOptions) { o.inOrder = true } }
func warnElementsIn() containsOption { return func(o *containsOptions) { o.warnElementsIn = true } }

func (t *T) containsExactlyElementsIn(expected starlark.Value, os ...containsOption) (starlark.Value, error) {
	opts := &containsOptions{}
	for _, o := range os {
		o(opts)
	}

	iterableActual, ok := t.actual.(starlark.Iterable)
	if !ok {
		return nil, t.failWithProposition("is not Iterable", "")
	}
	iterableExpected, ok := expected.(starlark.Iterable)
	if !ok {
		return nil, (&T{actual: expected}).failWithProposition("is not Iterable", "")
	}

	missing := newDuplicateCounter()
	extra := newDuplicateCounter()
	iterActual := iterableActual.Iterate()
	defer iterActual.Done()
	iterExpected := iterableExpected.Iterate()
	defer iterExpected.Done()

	warning := ""
	if opts.warnElementsIn {
		warning = "" +
			" Passing a single iterable to .containsExactly(*expected) is often" +
			" not the correct thing to do. Did you mean to call" +
			" .containsExactlyElementsIn(Iterable) instead?"
	}

	var elemActual, elemExpected starlark.Value
	iterations := 0
	for {
		// Step through both iterators comparing elements pairwise.
		if !iterActual.Next(&elemActual) {
			break
		}
		if !iterExpected.Next(&elemExpected) {
			extra.increment(elemActual)
			break
		}
		iterations += 1

		// As soon as we encounter a pair of elements that differ, we know that
		// inOrder cannot succeed, so we can check the rest of the elements
		// more normally. Since any previous pairs of elements we iterated
		// over were equal, they have no effect on the result now.
		ok, err := starlark.CompareDepth(syntax.NEQ, elemActual, elemExpected, maxdepth)
		if err != nil {
			return nil, err
		}
		if ok {
			// Missing elements; elements that are not missing will be removed.
			missing.increment(elemExpected)
			var m starlark.Value
			for iterExpected.Next(&m) {
				missing.increment(m)
			}

			// Remove all actual elements from missing, and add any that weren't
			// in missing to extra.
			if missing.contains(elemActual) {
				missing.decrement(elemActual)
			} else {
				extra.increment(elemActual)
			}
			var e starlark.Value
			for iterActual.Next(&e) {
				if missing.contains(e) {
					missing.decrement(e)
				} else {
					extra.increment(e)
				}
			}

			// Fail if there are either missing or extra elements.

			if !missing.empty() {
				if !extra.empty() {
					// Subject is missing required elements and has extra elements.
					msg := fmt.Sprintf("contains exactly <%s>."+
						" It is missing <%s> and has unexpected items <%s>",
						expected.String(), missing.String(), extra.String())
					return nil, t.failWithProposition(msg, warning)
				} else {
					return nil, t.failWithBadResults("contains exactly", expected,
						"is missing", missing, warning)
				}
			}

			if !extra.empty() {
				return nil, t.failWithBadResults("contains exactly", expected,
					"has unexpected items", extra, warning)
			}

			// The iterables were not in the same order.
			if opts.inOrder {
				return nil, t.failComparingValues(
					"contains exactly these elements in order", expected, "")
			}
		}
	}
	if iterations == 0 && missing.empty() && !extra.empty() {
		return nil, t.failWithProposition("is empty", "")
	}

	// We must have reached the end of one of the iterators without finding any
	// pairs of elements that differ. If the actual iterator still has elements,
	// they're extras. If the required iterator has elements, they're missing.
	var e starlark.Value
	for iterActual.Next(&e) {
		extra.increment(e)
	}
	if !extra.empty() {
		return nil, t.failWithBadResults("contains exactly", expected,
			"has unexpected items", extra, warning)
	}

	var m starlark.Value
	for iterExpected.Next(&m) {
		missing.increment(m)
	}
	if !missing.empty() {
		return nil, t.failWithBadResults("contains exactly", expected,
			"is missing", missing, warning)
	}

	// If neither iterator has elements, we reached the end and the elements
	// were in order.
	return starlark.None, nil
}

func named(t *T, args ...starlark.Value) (starlark.Value, error) {
	str, ok := args[0].(starlark.String)
	if !ok || str.Len() == 0 {
		return nil, errors.New(".named() expects a (non empty) string")
	}
	t.name = str.GoString()
	return t, nil
}

func isFalse(t *T, args ...starlark.Value) (starlark.Value, error) {
	if b, ok := t.actual.(starlark.Bool); ok && b == starlark.False {
		return starlark.None, nil
	}
	suffix := ""
	if !t.actual.Truth() {
		suffix = " However, it is falsy. Did you mean to call .isFalsy() instead?"
	}
	return nil, t.failWithProposition("is False", suffix)
}

func isFalsy(t *T, args ...starlark.Value) (starlark.Value, error) {
	if t.actual.Truth() {
		return nil, t.failWithProposition("is falsy", "")
	}
	return starlark.None, nil
}

func isTrue(t *T, args ...starlark.Value) (starlark.Value, error) {
	if b, ok := t.actual.(starlark.Bool); ok && b == starlark.True {
		return starlark.None, nil
	}
	suffix := ""
	if t.actual.Truth() {
		suffix = " However, it is truthy. Did you mean to call .isTruthy() instead?"
	}
	return nil, t.failWithProposition("is True", suffix)
}

func isTruthy(t *T, args ...starlark.Value) (starlark.Value, error) {
	if !t.actual.Truth() {
		return nil, t.failWithProposition("is truthy", "")
	}
	return starlark.None, nil
}

func (t *T) comparable(bName, verb string, op syntax.Token, other starlark.Value) (starlark.Value, error) {
	if err := t.failNone(bName, other); err != nil {
		return nil, err
	}
	ok, err := starlark.CompareDepth(op, t.actual, other, maxdepth)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, t.failComparingValues(verb, other, "")
	}
	return starlark.None, nil
}

func isAtLeast(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("isAtLeast", "is at least", syntax.LT, args[0])
}

func isAtMost(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("isAtMost", "is at most", syntax.GT, args[0])
}

func isGreaterThan(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("isGreaterThan", "is greater than", syntax.LE, args[0])
}

func isLessThan(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("isLessThan", "is less than", syntax.GE, args[0])
}
