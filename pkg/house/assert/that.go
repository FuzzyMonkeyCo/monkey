package assert

import (
	"fmt"
	"io"
	"sort"

	"go.starlark.net/starlark"
)

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

const maxdepth = 10

var (
	_ io.Closer         = (*T)(nil)
	_ starlark.Value    = (*T)(nil)
	_ starlark.HasAttrs = (*T)(nil)
)

type T struct {
	target starlark.Value
	closed bool
}

func That(target starlark.Value) *T {
	return &T{target: target}
}
func (t *T) Close() (err error) {
	if !t.closed {
		err = fmt.Errorf("well %+v", t)
	}
	return
}

func Starlark(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	//TODO: store closedness in thread?
	// Use bltn.Receiver() to check closedness
	var target starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &target); err != nil {
		return nil, err
	}
	return That(target), nil
}

func (t *T) String() string                           { return fmt.Sprintf("AssertThat(%s)", t.target.String()) }
func (t *T) Type() string                             { return "AssertThat" }
func (t *T) Freeze()                                  { t.target.Freeze() }
func (t *T) Truth() starlark.Bool                     { return t.target.Truth() }
func (t *T) Hash() (uint32, error)                    { return t.target.Hash() }
func (t *T) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name) }
func (t *T) AttrNames() []string                      { return attrNames }

var (
	methods = map[string]func(t *T, b *starlark.Builtin, args ...starlark.Tuple) (starlark.Value, error){
		"isAtMost": isAtMost,
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
		case "isAtMost":
			var arg1 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &arg1); err != nil {
				return nil, err
			}
			return method(t, b, arg1)
		}
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(t), nil
}

func isAtMost(t *T, b *starlark.Builtin, args ...starlark.Tuple) (starlark.Value, error) {
	other := args[0]
	if err := checkNone(b.Name(), other); err != nil {
		return nil, err
	}
	ok, err := starlark.CompareDepth(syntax.GT, t.actual, other, maxdepth)
	if err != nil {
		return nil, err
	}
	if ok {
		
	}
	//FIXME: typecheck + closedness (before+after call)
	x, _ := max.(starlark.Int).Uint64()
	y, _ := t.target.(starlark.Int).Uint64()
	if x < y {
		return nil, fmt.Errorf("%s: fails", b.Name())
	}
	return starlark.None, nil
}

// type T struct {
// 	emptySubject
// }

// AssertThat(actual).IsEqualTo(expected)
// AssertThat(actual).IsIn(expected_possibilities)
// assertThat(actual).containsExactly(64, 128, 256, 128).inOrder()

/// comparable :: lt + le + gt + ge
/// const +inf, -inf, nan

// func That(target starlark.Value) *T {
// 	switch {
// 	case isNumeric(target):
// 		return numericSubjet(target)
// 	case isComparable(target) && isIterable(target):
// 		return comparableIterableSubject(target)
// 	case isComparable(target):
// 		return comparableSubject(target)
// 	case isIterable(target):
// 		return iterableSubject(target)
// 	default:
// 		return defaultSubject(target)
// 	}
// }

// func isComparable(target starlark.Value) bool {
// 	_, ok := target.(starlark.Comparable)
// 	return ok
// }

// func isHashable(target starlark.Value) bool {
// 	_, err := target.Hash() // TODO: can panic on Float
// 	return err == nil
// }

// func isIterable(target starlark.Value) bool {
// 	_, ok := target.(starlark.Iterable)
// 	return ok
// }

// func isNumeric(target starlark.Value) bool {
// 	switch target.(type) {
// 	case starlark.Bool:
// 		return true
// 	case starlark.Int:
// 		return true
// 	case starlark.Float:
// 		return true
// 	default:
// 		return false
// 	}
// }

// func describeTimes(times int) string {
// 	if times == 1 {
// 		return "once"
// 	}
// 	return fmt.Sprintf("%d times", times)
// }

// type emptySubject struct {
// 	unresolvedSubjects map[T]struct{}
// 	actual starlark.Value
// 	name string
// 	// tStack = Python inspect.stack()
// }

// func (s *emptySubject) String() string {
//   // def __str__(self):
//   //   stack_iter = iter(self._stack)
//   //   for stack in stack_iter:
//   //     # Find the caller of AssertThat(...).
//   //     if stack[3] == 'AssertThat':
//   //       caller = next(stack_iter)
//   //       return ('{0}({1}) created in module {2}, line {3}, in {4}:\n'
//   //               '      {5}'
//   //               .format(self.__class__.__name__, self._GetSubject(),
//   //                       inspect.getmodulename(caller[1]),   # Module name.
//   //                       caller[2],                          # Line number.
//   //                       caller[3],                          # Function name.
//   //                       caller[4][0].strip()))              # Code snippet.
//   //   # The subject was not created by AssertThat().
//   //   return '{0}({1})'.format(self.__class__.__name__, self._GetSubject())
// 	return "TODO: func (s *emptySubject) String() string"
// }

// func (s *emptySubject) Named(name string) *emptySubject {
// 	s.name =name
// 	return s
// }

// func (s *emptySubject) name() string {
// 	return s.name
// }

// func (s *emptySubject) actualGet() starlark.Value {
// 	s.resolve()
// 	return actual()
// }

// func (s *emptySubject) actualSet(value starlark.Value) {
// 		s.actual=value
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

// func (s *emptySubject) resolveAll() {
// 	s.unresolvedSubjects = nil
// }

// func (s *emptySubject) getSubject
