package starlarktruth

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

var (
	_ starlark.Value    = (*T)(nil)
	_ starlark.HasAttrs = (*T)(nil)
)

func AssertThat(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	//TODO: store closedness in thread?
	// Use bltn.Receiver() to check closedness
	var target starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &target); err != nil {
		return nil, err
	}
	t := &T{
		actual: target,
	}
	return t, nil
}

func (t *T) String() string                           { return fmt.Sprintf("AssertThat(%s)", t.actual.String()) }
func (t *T) Type() string                             { return "AssertThat" }
func (t *T) Freeze()                                  { t.actual.Freeze() }
func (t *T) Truth() starlark.Bool                     { return t.actual.Truth() }
func (t *T) Hash() (uint32, error)                    { return t.actual.Hash() }
func (t *T) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name) }
func (t *T) AttrNames() []string                      { return attrNames }

var (
	methods = map[string]func(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error){
		"isAtLeast":     IsAtLeast,
		"isAtMost":      IsAtMost,
		"isGreaterThan": IsGreaterThan,
		"isLessThan":    IsLessThan,
	}

	attrNames = func() []string {
		names := make([]string, 0, len(methods))
		for name := range methods {
			names = append(names, name)
		}
		sort.Strings(names)
		return names
	}()
)

func builtinAttr(t *T, name string) (starlark.Value, error) {
	method := methods[name]
	if method == nil {
		return nil, nil // no such method
	}
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		switch b.Name() {
		// 1-arg attributes
		case "isAtLeast", "isAtMost", "isGreaterThan", "isLessThan":
			var arg1 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &arg1); err != nil {
				return nil, err
			}
			return method(t, b, arg1)
		default:
			return nil, fmt.Errorf("missing clause for attribute %q", b.Name())
		}
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(t), nil
}

// AssertThat(actual).IsEqualTo(expected)
// AssertThat(actual).IsIn(expected_possibilities)
// assertThat(actual).containsExactly(64, 128, 256, 128).inOrder()

/// comparable :: lt + le + gt + ge
/// const +inf, -inf, nan

// func describeTimes(times int) string {
// 	if times == 1 {
// 		return "once"
// 	}
// 	return fmt.Sprintf("%d times", times)
// }

// func (s *emptySubject) checkUnresolved() {
// 	if len(s.unresolvedSubjects) != 0 {
//         msg := []string{
// 			`The following assertions were unresolved. Perhaps you called`+
// 				` "AssertThat(thing.IsEmpty())" instead of`+
// 				` "AssertThat(thing).IsEmpty()".`,
// 		}
// 		//TODO: sort
// 		for u := range unresolvedSubjects {
// 			msg=append(msg,fmt.Sprintf(`    * %s`, u))
// 		}
// 		panic(strings.Join(msg, "\n")
// 		}
// 	}
// }
