package tags

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

var _ starlark.Unpacker = (*UniqueStrings)(nil)

// UniqueStrings implements starlark.Unpacker
type UniqueStrings struct {
	strings []string
}

// Unpack unmarshals UniqueStrings from a starlark.Value
func (us *UniqueStrings) Unpack(v starlark.Value) (err error) {
	us.strings, err = unpack(v)
	return
}

// GoStringsMap returns the unique strings as a Go map
func (us *UniqueStrings) GoStringsMap() (m Tags) {
	m = make(Tags, len(us.strings))
	for _, s := range us.strings {
		m[s] = struct{}{}
	}
	return
}

// GoStrings returns the list of unique strings as a Go slice
func (us *UniqueStrings) GoStrings() []string { return us.strings }

var _ starlark.Unpacker = (*UniqueStringsNonEmpty)(nil)

// UniqueStringsNonEmpty implements starlark.Unpacker
type UniqueStringsNonEmpty struct {
	strings []string
}

// Unpack unmarshals UniqueStringsNonEmpty from a starlark.Value
func (us *UniqueStringsNonEmpty) Unpack(v starlark.Value) (err error) {
	if us.strings, err = unpack(v); err != nil {
		return
	}
	if len(us.strings) == 0 {
		err = errors.New("must not be empty")
	}
	return
}

// GoStrings returns the list of unique strings as a Go slice
func (us *UniqueStringsNonEmpty) GoStrings() []string { return us.strings }

func unpack(v starlark.Value) ([]string, error) {
	list, ok := v.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("got %s, want list", v.Type())
	}

	l := list.Len()
	m := make(map[uint32]struct{}, l)
	strs := make([]string, 0, l)
	for i := 0; i < l; i++ {
		x := list.Index(i)

		s, isString := x.(starlark.String)
		if !isString {
			return nil, fmt.Errorf("got %s, want string", x.Type())
		}

		h, err := s.Hash()
		if err != nil {
			panic("unreachable")
		}
		if _, isDupe := m[h]; isDupe {
			return nil, fmt.Errorf("%s appears more than once", s)
		}
		m[h] = struct{}{}

		strs = append(strs, s.GoString())
	}
	return strs, nil
}
