package runtime

import (
	"fmt"
	"net/textproto"
	"slices"
	"sort"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// ctxHeader represents header data (e.g. HTTP headers) as a Starlark value for user assertions or mutation.
type ctxHeader struct {
	header textproto.MIMEHeader
	keys   []string

	frozen    bool
	itercount uint32 // number of active iterators (ignored if frozen)
}

var _ starlark.HasAttrs = (*ctxHeader)(nil)

func ctxHeaderAdd(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k, v starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &k, &v); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*ctxHeader)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	ch.header.Add(key, v.GoString())
	ch.keys = append(ch.keys, key)
	return starlark.None, nil
}

func ctxHeaderDel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*ctxHeader)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	ch.header.Del(key)
	ch.keys = slices.DeleteFunc(ch.keys, func(k string) bool { return k == key })
	return starlark.None, nil
}

func ctxHeaderGet(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*ctxHeader)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	return starlark.String(ch.header.Get(key)), nil
}

func ctxHeaderSet(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k, v starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &k, &v); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*ctxHeader)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	ch.header.Set(key, v.GoString())
	if !slices.Contains(ch.keys, key) {
		ch.keys = append(ch.keys, key)
	}
	return starlark.None, nil
}

func ctxHeaderValues(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*ctxHeader)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	return starlark.NewList(fromStrings(ch.header.Values(key))), nil
}

var ctxHeaderMethods = map[string]*starlark.Builtin{
	"add":    starlark.NewBuiltin("add", ctxHeaderAdd),
	"del":    starlark.NewBuiltin("del", ctxHeaderDel),
	"get":    starlark.NewBuiltin("get", ctxHeaderGet),
	"set":    starlark.NewBuiltin("set", ctxHeaderSet),
	"values": starlark.NewBuiltin("values", ctxHeaderValues),
}

func (ch *ctxHeader) Attr(name string) (starlark.Value, error) {
	switch name {
	case "add":
		if err := ch.checkMutable(name); err != nil {
			return nil, err
		}
		return ctxHeaderMethods[name].BindReceiver(ch), nil
	case "del":
		if err := ch.checkMutable(name); err != nil {
			return nil, err
		}
		return ctxHeaderMethods[name].BindReceiver(ch), nil
	case "get":
		return ctxHeaderMethods[name].BindReceiver(ch), nil
	case "set":
		if err := ch.checkMutable(name); err != nil {
			return nil, err
		}
		return ctxHeaderMethods[name].BindReceiver(ch), nil
	case "values":
		return ctxHeaderMethods[name].BindReceiver(ch), nil
	default:
		return nil, nil // no such method
	}
}

func (ch *ctxHeader) AttrNames() []string {
	names := make([]string, 0, len(ctxHeaderMethods))
	for name := range ctxHeaderMethods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func newCtxHeader(protoHeader []*fm.HeaderPair) *ctxHeader {
	ch := &ctxHeader{
		header: make(textproto.MIMEHeader, len(protoHeader)),
		keys:   make([]string, 0, len(protoHeader)),
	}
	for _, pair := range protoHeader {
		key := textproto.CanonicalMIMEHeaderKey(pair.GetKey())
		ch.keys = append(ch.keys, key)
		for _, value := range pair.GetValues() {
			ch.header.Add(key, value)
		}
	}
	// h := starlark.NewDict(len(protoHeader))
	// for _, kvs := range protoHeader {
	// 	key := starlark.String(kvs.GetKey())
	// 	values := kvs.GetValues()
	// 	vs := make([]starlark.Value, 0, len(values))
	// 	for _, value := range values {
	// 		vs = append(vs, starlark.String(value))
	// 	}
	// 	if err := h.SetKey(key, starlark.NewList(vs)); err != nil {
	// 		return nil, err
	// 	}
	// }
	// return &ctxHeader{header: h}, nil
	return ch
}

func (ch *ctxHeader) IntoProto() []*fm.HeaderPair {
	hs := make([]*fm.HeaderPair, 0, len(ch.keys))
	for _, key := range ch.keys {
		h := &fm.HeaderPair{
			Key:    key,
			Values: ch.header.Values(key),
		}
		hs = append(hs, h)
	}
	return hs
}

var _ starlark.IterableMapping = (*ctxHeader)(nil)

func (ch *ctxHeader) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", ch.Type()) }
func (ch *ctxHeader) String() string        { return ch.Type() }
func (ch *ctxHeader) Truth() starlark.Bool  { return true }
func (ch *ctxHeader) Type() string          { return "ctx_header" }

func (ch *ctxHeader) Freeze() {
	// ch.header.Freeze()
	ch.frozen = true
}

func (ch *ctxHeader) Get(x starlark.Value) (v starlark.Value, found bool, err error) {
	s, ok := x.(starlark.String)
	if !ok {
		return
	}
	vs := ch.header.Values(s.GoString())
	if vs == nil {
		return
	}
	return starlark.NewList(fromStrings(vs)), true, nil
}

func (ch *ctxHeader) Items() []starlark.Tuple {
	kvs := make([]starlark.Tuple, 0, len(ch.header))
	for _, key := range ch.keys {
		values := ch.header.Values(key)
		k := starlark.String(key)
		vs := starlark.NewList(fromStrings(values))
		kv := starlark.Tuple{k, vs}
		kvs = append(kvs, kv)
	}
	return kvs
}

func fromStrings(values []string) []starlark.Value {
	vs := make([]starlark.Value, 0, len(values))
	for _, v := range values {
		vs = append(vs, starlark.String(v))
	}
	return vs
}

func (ch *ctxHeader) Iterate() starlark.Iterator {
	if !ch.frozen {
		ch.itercount++
	}
	return &ctxHeaderIterator{ch: ch}
}

func (ch *ctxHeader) checkMutable(verb string) error {
	if ch.frozen {
		return fmt.Errorf("cannot %s frozen hash table", verb)
	}
	if ch.itercount > 0 {
		return fmt.Errorf("cannot %s hash table during iteration", verb)
	}
	return nil
}

type ctxHeaderIterator struct {
	ch *ctxHeader
	i  int
}

func (it *ctxHeaderIterator) Next(p *starlark.Value) bool {
	if it.i < len(it.ch.keys) {
		key := it.ch.keys[it.i]
		vs := fromStrings(it.ch.header.Values(key))
		*p = starlark.Tuple{starlark.String(key), starlark.NewList(vs)}
		it.i++
		return true
	}
	return false
}

func (it *ctxHeaderIterator) Done() {
	if !it.ch.frozen {
		it.ch.itercount--
	}
}
