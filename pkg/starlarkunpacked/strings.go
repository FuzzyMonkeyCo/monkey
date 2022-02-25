package starlarkunpacked

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Strings is a subtype of *starlark.List containing only starlark.String.s
type Strings struct {
	l *starlark.List
}

// *Strings implements everything that *starlark.List implements...
var (
	_ starlark.Comparable  = (*Strings)(nil)
	_ starlark.HasSetIndex = (*Strings)(nil)
	_ starlark.Sliceable   = (*Strings)(nil)
	_ starlark.HasAttrs    = (*Strings)(nil)
)

// ...and more!
var _ starlark.Unpacker = (*Strings)(nil)

func (sl *Strings) Unpack(v starlark.Value) error {
	list, ok := v.(*starlark.List)
	if !ok {
		return fmt.Errorf("got %s, want list", v.Type())
	}

	for i, l := 0, list.Len(); i < l; i++ {
		x := list.Index(i)
		if _, isString := x.(starlark.String); !isString {
			return fmt.Errorf("got %s, want string", x.Type())
		}
	}
	sl.l = list
	return nil
}

// GoStrings panics if any item is not a starlark.String
func (sl *Strings) GoStrings() (xs []string) {
	l := sl.l.Len()
	xs = make([]string, 0, l)
	for i := 0; i < l; i++ {
		xs = append(xs, sl.l.Index(i).(starlark.String).GoString())
	}
	return
}

func (sl *Strings) Append(v starlark.Value) error            { return sl.l.Append(v) }
func (sl *Strings) Attr(name string) (starlark.Value, error) { return sl.l.Attr(name) }
func (sl *Strings) AttrNames() []string                      { return sl.l.AttrNames() }
func (sl *Strings) Clear() error                             { return sl.l.Clear() }
func (sl *Strings) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	return sl.l.CompareSameType(op, y, depth)
}
func (sl *Strings) Freeze()                                   { sl.l.Freeze() }
func (sl *Strings) Hash() (uint32, error)                     { return sl.l.Hash() }
func (sl *Strings) Index(i int) starlark.Value                { return sl.l.Index(i) }
func (sl *Strings) Iterate() starlark.Iterator                { return sl.l.Iterate() }
func (sl *Strings) Len() int                                  { return sl.l.Len() }
func (sl *Strings) SetIndex(i int, v starlark.Value) error    { return sl.l.SetIndex(i, v) }
func (sl *Strings) Slice(start, end, step int) starlark.Value { return sl.l.Slice(start, end, step) }
func (sl *Strings) String() string                            { return sl.l.String() }
func (sl *Strings) Truth() starlark.Bool                      { return sl.l.Truth() }
func (sl *Strings) Type() string                              { return sl.l.Type() }
