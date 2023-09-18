package runtime

import (
	"fmt"
	"net/textproto"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// ctxHeader represents input request data as a Starlark value for user assertions or mutation.
type ctxHeader struct {
	//FIXME?
	header textproto.MIMEHeader
	keys   []string

	// header starlark.Value
	frozen    bool
	itercount uint32 // number of active iterators (ignored if frozen)
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

type ctxHeaderIterator struct {
	ch *ctxHeader
	i  int
}

func (it *ctxHeaderIterator) Next(p *starlark.Value) bool {
	if it.i < len(it.ch.keys) {
		key := it.ch.keys[it.i]
		vs := fromStrings(it.ch.header.Values(key))
		*p = starlark.NewList(vs)
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
