package starlarkunpacked

import (
	"fmt"

	"go.starlark.net/starlark"
)

// UniqueStrings implements starlark.Unpacker
type UniqueStrings struct {
	strings []string
}

var _ starlark.Unpacker = (*UniqueStrings)(nil)

// Unpack unmarshals UniqueStrings from a starlark.Value
func (us *UniqueStrings) Unpack(v starlark.Value) error {
	list, ok := v.(*starlark.List)
	if !ok {
		return fmt.Errorf("got %s, want list", v.Type())
	}

	l := list.Len()
	m := make(map[uint32]struct{}, l)
	us.strings = make([]string, 0, l)
	for i := 0; i < l; i++ {
		x := list.Index(i)

		s, isString := x.(starlark.String)
		if !isString {
			return fmt.Errorf("got %s, want string", x.Type())
		}

		h, err := s.Hash()
		if err != nil {
			panic("unreachable")
		}
		if _, isDupe := m[h]; isDupe {
			return fmt.Errorf("%s appears more than once", s)
		}
		m[h] = struct{}{}

		us.strings = append(us.strings, s.GoString())
	}
	return nil
}

// GoStrings returns the list of unique strings as a Go slice
func (us *UniqueStrings) GoStrings() []string {
	return us.strings
}
