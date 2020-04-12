package starlarktruth

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

const module = "AssertThat"

var (
	_ starlark.Value    = (*T)(nil)
	_ starlark.HasAttrs = (*T)(nil)
)

// NewModule registers a Starlark module of https://truth.dev/
func NewModule(predeclared starlark.StringDict) {
	predeclared[module] = starlark.NewBuiltin(module, func(
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
}

func newT(target starlark.Value) *T { return &T{actual: target} }

func (t *T) String() string                           { return fmt.Sprintf("%s(%s)", module, t.actual.String()) }
func (t *T) Type() string                             { return module }
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
		"containsNoDuplicates": containsNoDuplicates,
		"isCallable":           isCallable,
		"isEmpty":              isEmpty,
		"isFalse":              isFalse,
		"isFalsy":              isFalsy,
		"isNone":               isNone,
		"isNotCallable":        isNotCallable,
		"isNotEmpty":           isNotEmpty,
		"isNotNone":            isNotNone,
		"isTrue":               isTrue,
		"isTruthy":             isTruthy,
	}

	methods1arg = attrs{
		"contains":                         contains,
		"containsAllIn":                    containsAllIn,
		"containsAllInOrderIn":             containsAllInOrderIn,
		"containsExactlyElementsIn":        containsExactlyElementsIn,
		"containsExactlyElementsInOrderIn": containsExactlyElementsInOrderIn,
		"containsExactlyItemsIn":           containsExactlyItemsIn,
		"doesNotContain":                   doesNotContain,
		"doesNotHaveAttribute":             doesNotHaveAttribute,
		"hasAttribute":                     hasAttribute,
		"hasSize":                          hasSize,
		"isAtLeast":                        isAtLeast,
		"isAtMost":                         isAtMost,
		"isEqualTo":                        isEqualTo,
		"isGreaterThan":                    isGreaterThan,
		"isIn":                             isIn,
		"isLessThan":                       isLessThan,
		"isNotEqualTo":                     isNotEqualTo,
		"isNotIn":                          isNotIn,
		"named":                            named,
	}

	methodsNargs = attrs{
		"containsAllOf":          containsAllOf,
		"containsAllOfInOrder":   containsAllOfInOrder,
		"containsExactly":        containsExactly,
		"containsExactlyInOrder": containsExactlyInOrder,
		"isAnyOf":                isAnyOf,
		"isNoneOf":               isNoneOf,
	}

	methods = []attrs{
		methods0args,
		methods1arg,
	}

	attrNames = func() []string {
		names := make([]string, 0, len(methods1arg))
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
	for i, ms := range methods {
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
			thread.Print(thread, fmt.Sprintf(">>> closeness = %d", c))
			closeness = c
		}
		defer thread.SetLocal("closeness", 1+closeness)
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: unexpected keyword arguments", b.Name())
		}
		switch nArgs {
		case -1:
			ret, err := method(t, []starlark.Value(args)...)
			switch err {
			case nil:
				return ret, nil
			case errUnhandled:
				return nil, t.unhandled(b.Name(), args)
			default:
				return nil, err
			}
		case 0:
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
				return nil, err
			}
			ret, err := method(t)
			switch err {
			case nil:
				return ret, nil
			case errUnhandled:
				return nil, t.unhandled(b.Name(), args)
			default:
				return nil, err
			}
		case 1:
			var arg1 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &arg1); err != nil {
				return nil, err
			}
			ret, err := method(t, arg1)
			switch err {
			case nil:
				return ret, nil
			case errUnhandled:
				return nil, t.unhandled(b.Name(), args)
			default:
				return nil, err
			}
		default:
			panic("unreachable")
		}
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(t), nil
}
