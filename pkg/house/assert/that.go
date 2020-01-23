package assert

import (
	"fmt"

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

var (
	_ starlark.Value    = (*T)(nil)
	_ starlark.HasAttrs = (*T)(nil)
	// _ starlark.HasSetKey       = (*modelState)(nil)
	// _ starlark.IterableMapping = (*modelState)(nil)
	// _ starlark.Sequence        = (*modelState)(nil)
	// _ starlark.Comparable      = (*modelState)(nil)
)

type T struct{}

// func newModelState(size int) *modelState {
// 	return &modelState{d: starlark.NewDict(size)}
// }
// func (s *modelState) Clear() error { return s.d.Clear() }
// func (s *modelState) Delete(k starlark.Value) (starlark.Value, bool, error) {
// 	if err := slValuePrintableASCII(k); err != nil {
// 		return nil, false, err
// 	}
// 	log.Printf("[NFO] Delete(%v)", k)
// 	return s.d.Delete(k)
// }
// func (s *modelState) Get(k starlark.Value) (starlark.Value, bool, error) {
// 	if err := slValuePrintableASCII(k); err != nil {
// 		return nil, false, err
// 	}
// 	log.Printf("[NFO] Get(%v)", k)
// 	return s.d.Get(k)
// }
// func (s *modelState) Items() []starlark.Tuple    { return s.d.Items() }
// func (s *modelState) Keys() []starlark.Value     { return s.d.Keys() }
// func (s *modelState) Len() int                   { return s.d.Len() }
// func (s *modelState) Iterate() starlark.Iterator { return s.d.Iterate() }
// func (s *modelState) SetKey(k, v starlark.Value) error {
// 	if err := slValuePrintableASCII(k); err != nil {
// 		return err
// 	}
// 	log.Printf("[NFO] SetKey(%v, %v)", k, v)
// 	return s.d.SetKey(k, v)
// }
// func (s *modelState) String() string                           { return s.d.String() }
// func (s *modelState) Type() string                             { return "ModelState" }
// func (s *modelState) Freeze()                                  { s.d.Freeze() }
// func (s *modelState) Truth() starlark.Bool                     { return s.d.Truth() }
// func (s *modelState) Hash() (uint32, error)                    { return s.d.Hash() }
// func (s *modelState) Attr(name string) (starlark.Value, error) { return s.d.Attr(name) }
// func (s *modelState) AttrNames() []string                      { return s.d.AttrNames() }
// func (s *modelState) CompareSameType(op syntax.Token, ss starlark.Value, depth int) (bool, error) {
// 	return s.d.CompareSameType(op, ss, depth)
// }

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
