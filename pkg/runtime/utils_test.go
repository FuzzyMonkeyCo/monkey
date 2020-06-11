package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestStarlarkValueClone(t *testing.T) {
	type testcase struct {
		value starlark.Value
		edit  func(starlark.Value)
	}

	for someCaseName, someCase := range map[string]*testcase{
		"replace an item of a list within a tuple": {
			value: starlark.Tuple([]starlark.Value{
				starlark.String("blip"),
				starlark.Tuple([]starlark.Value{starlark.String("blop")}),
				starlark.NewList([]starlark.Value{starlark.String("blap")}),
			}),
			edit: func(v starlark.Value) {
				t := v.(starlark.Tuple)
				vv := t[2]
				ll := vv.(*starlark.List)
				ll.SetIndex(0, starlark.Bool(true))
			},
		},
		"replace an item of a list within a list": {
			value: starlark.NewList([]starlark.Value{
				starlark.String("blip"),
				starlark.Tuple([]starlark.Value{starlark.String("blop")}),
				starlark.NewList([]starlark.Value{starlark.String("blap")}),
			}),
			edit: func(v starlark.Value) {
				l := v.(*starlark.List)
				vv := l.Index(2)
				ll := vv.(*starlark.List)
				ll.SetIndex(0, starlark.Bool(true))
			},
		},
		"delete a value of a dict": {
			value: func() starlark.Value {
				someDict := starlark.NewDict(2)
				someDict.SetKey(starlark.String("key"), starlark.String("value"))
				someOtherDict := starlark.NewDict(3)
				someOtherDict.SetKey(starlark.String("a"), starlark.Bool(true))
				someOtherDict.SetKey(starlark.String("b"), starlark.MakeInt(42))
				someOtherDict.SetKey(starlark.String("c"), starlark.Float(4.2))
				someDict.SetKey(starlark.String("k"), someOtherDict)
				return someDict
			}(),
			edit: func(v starlark.Value) {
				d := v.(*starlark.Dict)
				_, found, err := d.Delete(starlark.String("key"))
				require.NoError(t, err)
				require.True(t, found)
			},
		},
		"delete a value of a dict within a dict": {
			value: func() starlark.Value {
				someDict := starlark.NewDict(2)
				someDict.SetKey(starlark.String("key"), starlark.String("value"))
				someOtherDict := starlark.NewDict(3)
				someOtherDict.SetKey(starlark.String("a"), starlark.Bool(true))
				someOtherDict.SetKey(starlark.String("b"), starlark.MakeInt(42))
				someOtherDict.SetKey(starlark.String("c"), starlark.Float(4.2))
				someDict.SetKey(starlark.String("k"), someOtherDict)
				return someDict
			}(),
			edit: func(v starlark.Value) {
				d := v.(*starlark.Dict)
				vv, found, err := d.Get(starlark.String("k"))
				require.NoError(t, err)
				require.True(t, found)
				dd := vv.(*starlark.Dict)
				_, found, err = dd.Delete(starlark.String("c"))
				require.NoError(t, err)
				require.True(t, found)
			},
		},
	} {
		t.Run(someCaseName, func(t *testing.T) {
			repr := someCase.value.String()
			require.NotEmpty(t, repr)
			cloned, err := slValueCopy(someCase.value)
			require.NoError(t, err)
			t.Logf("A cloned value shares its representation")
			require.Equal(t, repr, cloned.String())

			someCase.edit(cloned)
			t.Logf("Editing a cloned value changes its representation...")
			require.NotEqual(t, repr, cloned.String())
			t.Logf("... but does not change the original value's representation.")
			require.Equal(t, repr, someCase.value.String())
		})
	}
}
