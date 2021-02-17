package tags

import (
	"fmt"

	"go.starlark.net/starlark"
)

var _ starlark.Unpacker = (*StarlarkStringList)(nil)

// StarlarkStringList implements starlark.Unpacker
type StarlarkStringList struct {
	Uniques Tags
}

// Unpack unmarshals StarlarkStringList from a starlark.Value
func (sl *StarlarkStringList) Unpack(v starlark.Value) error {
	list, ok := v.(*starlark.List)
	if !ok {
		return fmt.Errorf("got %s, want list", v.Type())
	}

	sl.Uniques = make(Tags, list.Len())
	it := list.Iterate()
	defer it.Done()
	var x starlark.Value
	for it.Next(&x) {
		str, ok := starlark.AsString(x)
		if !ok {
			return fmt.Errorf("got %s, want string", x.Type())
		}
		if err := LegalName(str); err != nil {
			return err
		}
		if _, ok := sl.Uniques[str]; ok {
			return fmt.Errorf("string %s appears at least twice in list", x.String())
		}
		sl.Uniques[str] = struct{}{}
	}
	return nil
}
