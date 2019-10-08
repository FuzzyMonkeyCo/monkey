package pkg

import (
	"fmt"
	"reflect"
	"unicode"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var (
	_ starlark.Value           = (*modelState)(nil)
	_ starlark.HasAttrs        = (*modelState)(nil)
	_ starlark.HasSetKey       = (*modelState)(nil)
	_ starlark.IterableMapping = (*modelState)(nil)
	_ starlark.Sequence        = (*modelState)(nil)
	_ starlark.Comparable      = (*modelState)(nil)
)

type modelState struct {
	d *starlark.Dict
}

func newModelState(size int) *modelState {
	return &modelState{d: starlark.NewDict(size)}
}
func (s *modelState) Clear() error { return s.d.Clear() }
func (s *modelState) Delete(k starlark.Value) (starlark.Value, bool, error) {
	if err := slValuePrintableASCII(k); err != nil {
		return nil, false, err
	}
	return s.d.Delete(k)
}
func (s *modelState) Get(k starlark.Value) (starlark.Value, bool, error) {
	if err := slValuePrintableASCII(k); err != nil {
		return nil, false, err
	}
	return s.d.Get(k)
}
func (s *modelState) Items() []starlark.Tuple    { return s.d.Items() }
func (s *modelState) Keys() []starlark.Value     { return s.d.Keys() }
func (s *modelState) Len() int                   { return s.d.Len() }
func (s *modelState) Iterate() starlark.Iterator { return s.d.Iterate() }
func (s *modelState) SetKey(k, v starlark.Value) error {
	if err := slValuePrintableASCII(k); err != nil {
		return err
	}
	return s.d.SetKey(k, v)
}
func (s *modelState) String() string                           { return s.d.String() }
func (s *modelState) Type() string                             { return "ModelState" }
func (s *modelState) Freeze()                                  { s.d.Freeze() }
func (s *modelState) Truth() starlark.Bool                     { return s.d.Truth() }
func (s *modelState) Hash() (uint32, error)                    { return s.d.Hash() }
func (s *modelState) Attr(name string) (starlark.Value, error) { return s.d.Attr(name) }
func (s *modelState) AttrNames() []string                      { return s.d.AttrNames() }
func (s *modelState) CompareSameType(op syntax.Token, ss starlark.Value, depth int) (bool, error) {
	return s.d.CompareSameType(op, ss, depth)
}

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

func slValueFromInterface(x interface{}) (starlark.Value, error) {
	if x == nil {
		return starlark.None, nil
	}
	switch v := reflect.ValueOf(x); v.Kind() {
	case reflect.Bool:
		return starlark.Bool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		return starlark.MakeInt64(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return starlark.MakeUint64(v.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return starlark.Float(v.Float()), nil
	case reflect.String:
		return starlark.String(v.String()), nil
	case reflect.Slice:
		values := make([]starlark.Value, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			s, err := slValueFromInterface(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			values = append(values, s)
		}
		return starlark.NewList(values), nil
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("expected string keys: %T", x)
		}
		values := starlark.NewDict(v.Len())
		for _, k := range v.MapKeys() {
			value := v.MapIndex(k)
			key := k.String()
			if err := printableASCII(key); err != nil {
				return nil, errors.Wrap(err, "illegal string key")
			}
			s, err := slValueFromInterface(value.Interface())
			if err != nil {
				return nil, err
			}
			if err = values.SetKey(starlark.String(key), s); err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		err := fmt.Errorf("not a JSON value: %T %+v", x, x)
		return nil, err
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
			if err = vs.SetKey(k, v); err != nil {
				return
			}
		}
		dst = vs
		return
	case *modelState:
		vs := newModelState(v.Len())
		for _, kv := range v.Items() {
			k, v := kv.Index(0), kv.Index(1)
			if err = vs.SetKey(k, v); err != nil {
				return
			}
		}
		dst = vs
		return
	default:
		// TODO: case *starlark.Set:
		err = fmt.Errorf("unexpected %T: %+v", src, src)
		return
	}
}
