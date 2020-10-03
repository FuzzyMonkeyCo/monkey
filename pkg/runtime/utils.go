package runtime

import (
	"fmt"
	"unicode"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

func slValuePrintableASCII(k starlark.Value) error {
	key, ok := k.(starlark.String)
	if !ok {
		return fmt.Errorf("expected a string, got: %s", k.Type())
	}
	return printableASCII(key.GoString())
}

func printableASCII(s string) error {
	l := 0
	for _, c := range s {
		if !(c <= unicode.MaxASCII && unicode.IsPrint(c)) {
			return fmt.Errorf("string contains non-ASCII or non-printable characters: %q", s)
		}
		l++
	}
	if l > 255 {
		return fmt.Errorf("string must be shorter than 256 characters: %q", s)
	}
	return nil
}

func slValueIsProtoable(value starlark.Value) (err error) {
	switch v := value.(type) {
	case starlark.NoneType:
		return
	case starlark.Bool:
		return
	case starlark.Int:
		return
	case starlark.Float:
		return
	case starlark.String:
		return
	case *starlark.List:
		for i := 0; i < v.Len(); i++ {
			if err = slValueIsProtoable(v.Index(i)); err != nil {
				return
			}
		}
		return
	case starlark.Tuple:
		for i := 0; i < v.Len(); i++ {
			if err = slValueIsProtoable(v.Index(i)); err != nil {
				return
			}
		}
		return
	case *starlark.Dict:
		for _, kv := range v.Items() {
			if err = slValuePrintableASCII(kv.Index(0)); err != nil {
				return
			}
			if err = slValueIsProtoable(kv.Index(1)); err != nil {
				return
			}
		}
		return
	case *modelState:
		for _, kv := range v.Items() {
			if err = slValuePrintableASCII(kv.Index(0)); err != nil {
				return
			}
			if err = slValueIsProtoable(kv.Index(1)); err != nil {
				return
			}
		}
		return
	default:
		err = fmt.Errorf("unexpected %T: %s", value, value.String())
		return
	}
}

func slValueFromProto(value *types.Value, parent *modelState) (starlark.Value, error) {
	switch value.GetKind().(type) {
	case *types.Value_NullValue:
		return starlark.None, nil
	case *types.Value_BoolValue:
		return starlark.Bool(value.GetBoolValue()), nil
	case *types.Value_NumberValue:
		return starlark.Float(value.GetNumberValue()), nil
	case *types.Value_StringValue:
		return starlark.String(value.GetStringValue()), nil
	case *types.Value_ListValue:
		values := value.GetListValue().GetValues()
		vals := make([]starlark.Value, 0, len(values))
		for _, v := range values {
			val, err := slValueFromProto(v, parent)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
		return starlark.NewList(vals), nil
	case *types.Value_StructValue:
		values := value.GetStructValue().GetFields()
		vals := newModelState(len(values), parent)
		for key, v := range values {
			if err := printableASCII(key); err != nil {
				return nil, errors.Wrap(err, "illegal string key")
			}
			val, err := slValueFromProto(v, vals)
			if err != nil {
				return nil, err
			}
			if err = vals.SetKey(starlark.String(key), val); err != nil {
				return nil, err
			}
		}
		return vals, nil
	default:
		panic(fmt.Errorf("unhandled: %T %+v", value, value))
	}
}

func slValueCopy(src starlark.Value) (dst starlark.Value, err error) {
	switch v := src.(type) {
	case starlark.NoneType:
		dst = starlark.None
		return
	case starlark.Bool:
		dst = v
		return
	case starlark.Int:
		dst = starlark.MakeBigInt(v.BigInt())
		return
	case starlark.Float:
		dst = v
		return
	case starlark.String:
		dst = starlark.String(v.GoString())
		return
	case *starlark.List:
		vs := make([]starlark.Value, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			var vv starlark.Value
			if vv, err = slValueCopy(v.Index(i)); err != nil {
				return
			}
			vs = append(vs, vv)
		}
		dst = starlark.NewList(vs)
		return
	case starlark.Tuple:
		vs := make([]starlark.Value, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			var vv starlark.Value
			if vv, err = slValueCopy(v.Index(i)); err != nil {
				return
			}
			vs = append(vs, vv)
		}
		dst = starlark.Tuple(vs)
		return
	case *starlark.Dict:
		vs := starlark.NewDict(v.Len())
		for _, kv := range v.Items() {
			k, v := kv.Index(0), kv.Index(1)
			if err = slValuePrintableASCII(k); err != nil {
				return
			}
			var kk, vv starlark.Value
			if kk, err = slValueCopy(k); err != nil {
				return
			}
			if vv, err = slValueCopy(v); err != nil {
				return
			}
			if err = vs.SetKey(kk, vv); err != nil {
				return
			}
		}
		dst = vs
		return
	case *modelState:
		vs := newModelState(v.Len(), v.parent) //FIXME? v.parent
		for _, kv := range v.Items() {
			k, v := kv.Index(0), kv.Index(1)
			var kk, vv starlark.Value
			if kk, err = slValueCopy(k); err != nil {
				return
			}
			if vv, err = slValueCopy(v); err != nil {
				return
			}
			// Key is checked by custom SetKey
			if err = vs.SetKey(kk, vv); err != nil {
				return
			}
		}
		dst = vs
		return
	default:
		err = fmt.Errorf("unexpected %T: %+v", src, src)
		return
	}
}
