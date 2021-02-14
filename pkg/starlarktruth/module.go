package starlarktruth

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

// Default is the default module constructor. It is merely the string "assert".
const Default = "assert"

type themodule struct{}

var _ starlark.Value = (*themodule)(nil)
var _ starlark.HasAttrs = (*themodule)(nil)

// NewModule registers a Starlark module of https://truth.dev/
func NewModule(predeclared starlark.StringDict) { predeclared[Default] = &themodule{} }

func (m *themodule) String() string        { return Default }
func (m *themodule) Type() string          { return Default }
func (m *themodule) Freeze()               {}
func (m *themodule) Truth() starlark.Bool  { return false }
func (m *themodule) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: %s", Default) }
func (m *themodule) AttrNames() []string   { return []string{"that"} }
func (m *themodule) Attr(name string) (starlark.Value, error) {
	if name != "that" {
		return nil, nil // no such method
	}
	b := starlark.NewBuiltin(Default, func(
		thread *starlark.Thread,
		b *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		//TODO: store closedness in thread? bltn.Receiver() to check closedness
		var target starlark.Value
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &target); err != nil {
			return nil, err
		}
		return newT(target), nil
	})
	return b.BindReceiver(m), nil
}

var _ starlark.Value = (*T)(nil)
var _ starlark.HasAttrs = (*T)(nil)

func newT(target starlark.Value) *T { return &T{actual: target} }

func (t *T) String() string                           { return fmt.Sprintf("%s(%s)", Default, t.actual.String()) }
func (t *T) Type() string                             { return Default }
func (t *T) Freeze()                                  { t.actual.Freeze() }
func (t *T) Truth() starlark.Bool                     { return t.actual.Truth() }
func (t *T) Hash() (uint32, error)                    { return t.actual.Hash() }
func (t *T) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name) }
func (t *T) AttrNames() []string                      { return attrNames }

type (
	attr  func(t *T, args ...starlark.Value) (starlark.Value, error)
	attrs map[string]attr
)

// TODO: turn all builtins matching *InOrder* into closedness-aware .inOrder()s

var (
	methods0args = attrs{
		"contains_no_duplicates": containsNoDuplicates,
		"is_callable":            isCallable,
		"is_empty":               isEmpty,
		"is_false":               isFalse,
		"is_falsy":               isFalsy,
		"is_none":                isNone,
		"is_not_callable":        isNotCallable,
		"is_not_empty":           isNotEmpty,
		"is_not_none":            isNotNone,
		"is_ordered":             isOrdered,
		"is_strictly_ordered":    isStrictlyOrdered,
		"is_true":                isTrue,
		"is_truthy":              isTruthy,
	}

	methods1arg = attrs{
		"contains":                              contains,
		"contains_all_in":                       containsAllIn,
		"contains_all_in_order_in":              containsAllInOrderIn,
		"contains_any_in":                       containsAnyIn,
		"contains_exactly_elements_in":          containsExactlyElementsIn,
		"contains_exactly_elements_in_order_in": containsExactlyElementsInOrderIn,
		"contains_exactly_items_in":             containsExactlyItemsIn,
		"contains_key":                          containsKey,
		"contains_match":                        containsMatch,
		"contains_none_in":                      containsNoneIn,
		"does_not_contain":                      doesNotContain,
		"does_not_contain_key":                  doesNotContainKey,
		"does_not_contain_match":                doesNotContainMatch,
		"does_not_have_attribute":               doesNotHaveAttribute,
		"does_not_match":                        doesNotMatch,
		"ends_with":                             endsWith,
		"has_attribute":                         hasAttribute,
		"has_length":                            hasLength,
		"has_size":                              hasSize,
		"is_at_least":                           isAtLeast,
		"is_at_most":                            isAtMost,
		"is_equal_to":                           isEqualTo,
		"is_greater_than":                       isGreaterThan,
		"is_in":                                 isIn,
		"is_less_than":                          isLessThan,
		"is_not_equal_to":                       isNotEqualTo,
		"is_not_in":                             isNotIn,
		"is_ordered_according_to":               isOrderedAccordingTo,
		"is_strictly_ordered_according_to":      isStrictlyOrderedAccordingTo,
		"matches":                               matches,
		"named":                                 named,
		"starts_with":                           startsWith,
	}

	methods2args = attrs{
		"contains_item":         containsItem,
		"does_not_contain_item": doesNotContainItem,
	}

	methodsNargs = attrs{
		"contains_all_of":           containsAllOf,
		"contains_all_of_in_order":  containsAllOfInOrder,
		"contains_any_of":           containsAnyOf,
		"contains_exactly":          containsExactly,
		"contains_exactly_in_order": containsExactlyInOrder,
		"contains_none_of":          containsNoneOf,
		"is_any_of":                 isAnyOf,
		"is_none_of":                isNoneOf,
	}

	methods = []attrs{
		methodsNargs,
		methods0args,
		methods1arg,
		methods2args,
	}

	attrNames = func() []string {
		count := 0
		for _, ms := range methods {
			count += len(ms)
		}
		names := make([]string, 0, count)
		for _, ms := range methods {
			for name := range ms {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		return names
	}()
)

func findAttr(name string) (attr, int) {
	for i, ms := range methods[1:] {
		if m, ok := ms[name]; ok {
			return m, i
		}
	}
	if m, ok := methodsNargs[name]; ok {
		return m, -1
	}
	return nil, 0
}

func builtinAttr(t *T, name string) (starlark.Value, error) {
	method, nArgs := findAttr(name)
	if method == nil {
		return nil, nil // no such method
	}
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		closeness := 0
		if c, ok := thread.Local("closeness").(int); ok {
			// thread.Print(thread, fmt.Sprintf(">>> closeness = %d", c))
			closeness = c
		}
		defer thread.SetLocal("closeness", 1+closeness)

		if err := t.registerValues(thread); err != nil {
			return nil, err
		}

		var argz []starlark.Value
		switch nArgs {
		case -1:
			if len(kwargs) > 0 {
				return nil, fmt.Errorf("%s: unexpected keyword arguments", b.Name())
			}
			argz = []starlark.Value(args)
		case 0:
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, nArgs); err != nil {
				return nil, err
			}
		case 1:
			var arg1 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, nArgs, &arg1); err != nil {
				return nil, err
			}
			argz = []starlark.Value{arg1}
		case 2:
			var arg1, arg2 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, nArgs, &arg1, &arg2); err != nil {
				return nil, err
			}
			argz = []starlark.Value{arg1, arg2}
		default:
			err := fmt.Errorf("unexpected #args for %s.that(%s).%q(): %d", Default, t.actual.String(), name, nArgs)
			return nil, err
		}

		ret, err := method(t, argz...)
		switch err {
		case nil:
			return ret, nil
		case errUnhandled:
			return nil, t.unhandled(b.Name(), argz...)
		default:
			return nil, err
		}
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(t), nil
}
