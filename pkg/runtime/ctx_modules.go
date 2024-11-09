package runtime

import (
	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// TODO: rename `ctx` to `cx`

func headerPairs(protoHeaders []*fm.HeaderPair) (starlark.Value, error) {
	d := starlark.NewDict(len(protoHeaders)) //fixme: dont make a dict out of repeated HeaderPair.s

	for _, kvs := range protoHeaders {
		values := kvs.GetValues()
		vs := make([]starlark.Value, 0, len(values))
		for _, value := range values {
			vs = append(vs, starlark.String(value))
		}
		if err := d.SetKey(starlark.String(kvs.GetKey()), starlark.NewList(vs)); err != nil {
			return nil, err
		}
	}
	return d, nil
}
