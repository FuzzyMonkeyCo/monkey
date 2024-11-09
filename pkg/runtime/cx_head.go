package runtime

import (
	"fmt"
	"net/textproto"
	"slices"
	"sort"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// cxHead represents header data (e.g. HTTP headers) as a Starlark value for user assertions or mutation.
// Should be accessible through checks under `ctx.request.headers` and/or `ctx.respones.headers`.
type cxHead struct {
	header textproto.MIMEHeader
	keys   []string

	frozen    bool
	itercount uint32 // number of active iterators (ignored if frozen)
}

func newcxHead(protoHeader []*fm.HeaderPair) *cxHead {
	ch := &cxHead{
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
	return ch
}

var _ starlark.HasAttrs = (*cxHead)(nil)

func cxHeadAdd(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k, v starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &k, &v); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*cxHead)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	ch.header.Add(key, v.GoString())
	ch.keys = append(ch.keys, key)
	return starlark.None, nil
}

func cxHeadDel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*cxHead)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	ch.header.Del(key)
	ch.keys = slices.DeleteFunc(ch.keys, func(k string) bool { return k == key })
	return starlark.None, nil
}

func cxHeadGet(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*cxHead)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	return starlark.String(ch.header.Get(key)), nil
}

func cxHeadSet(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k, v starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &k, &v); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*cxHead)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	ch.header.Set(key, v.GoString())
	if !slices.Contains(ch.keys, key) {
		ch.keys = append(ch.keys, key)
	}
	return starlark.None, nil
}

func cxHeadValues(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k); err != nil {
		return nil, err
	}
	ch := b.Receiver().(*cxHead)
	key := textproto.CanonicalMIMEHeaderKey(k.GoString())
	return starlark.NewList(fromStrings(ch.header.Values(key))), nil
}

var cxHeadMethods = map[string]*starlark.Builtin{
	"add":    starlark.NewBuiltin("add", cxHeadAdd),
	"del":    starlark.NewBuiltin("del", cxHeadDel),
	"get":    starlark.NewBuiltin("get", cxHeadGet),
	"set":    starlark.NewBuiltin("set", cxHeadSet),
	"values": starlark.NewBuiltin("values", cxHeadValues),
}

func (ch *cxHead) Attr(name string) (starlark.Value, error) {
	switch name {
	case "add":
		if err := ch.checkMutable(name); err != nil {
			return nil, err
		}
		return cxHeadMethods[name].BindReceiver(ch), nil
	case "del":
		if err := ch.checkMutable(name); err != nil {
			return nil, err
		}
		return cxHeadMethods[name].BindReceiver(ch), nil
	case "get":
		return cxHeadMethods[name].BindReceiver(ch), nil
	case "set":
		if err := ch.checkMutable(name); err != nil {
			return nil, err
		}
		return cxHeadMethods[name].BindReceiver(ch), nil
	case "values":
		return cxHeadMethods[name].BindReceiver(ch), nil
	default:
		return nil, nil // no such method
	}
}

func (ch *cxHead) AttrNames() []string {
	names := make([]string, 0, len(cxHeadMethods))
	for name := range cxHeadMethods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (ch *cxHead) IntoProto() []*fm.HeaderPair {
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

var _ starlark.IterableMapping = (*cxHead)(nil)

func (ch *cxHead) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", ch.Type()) }
func (ch *cxHead) String() string        { return ch.Type() }
func (ch *cxHead) Truth() starlark.Bool  { return true }
func (ch *cxHead) Type() string          { return "ctx_headers" }
func (ch *cxHead) Freeze()               { ch.frozen = true }

func (ch *cxHead) Get(x starlark.Value) (v starlark.Value, found bool, err error) {
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

func (ch *cxHead) Items() []starlark.Tuple {
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

func (ch *cxHead) Iterate() starlark.Iterator {
	if !ch.frozen {
		ch.itercount++
	}
	return &cxHeadIterator{ch: ch}
}

func (ch *cxHead) checkMutable(verb string) error {
	if ch.frozen {
		return fmt.Errorf("cannot %s frozen hash table", verb)
	}
	if ch.itercount > 0 {
		return fmt.Errorf("cannot %s hash table during iteration", verb)
	}
	return nil
}

type cxHeadIterator struct {
	ch *cxHead
	i  int
}

func (it *cxHeadIterator) Next(p *starlark.Value) bool {
	if it.i < len(it.ch.keys) {
		key := it.ch.keys[it.i]
		vs := fromStrings(it.ch.header.Values(key))
		*p = starlark.Tuple{starlark.String(key), starlark.NewList(vs)}
		it.i++
		return true
	}
	return false
}

func (it *cxHeadIterator) Done() {
	if !it.ch.frozen {
		it.ch.itercount--
	}
}
