package runtime

import (
	"fmt"
	"log"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// statedict

var (
	_ starlark.Value           = (*modelState)(nil)
	_ starlark.HasAttrs        = (*modelState)(nil)
	_ starlark.HasSetKey       = (*modelState)(nil)
	_ starlark.IterableMapping = (*modelState)(nil)
	_ starlark.Sequence        = (*modelState)(nil)
	_ starlark.Comparable      = (*modelState)(nil)

	olddict   *starlark.Builtin
	datapaths = &dataPaths{}

	// Addressable constants

	statedictCallResponse = &modelState{}
	statedictModel        = &modelState{}
)

type dataPaths struct {
	current []starlark.Value
	ks      []string
	vs      []starlark.Value
}

type modelState struct {
	d      *starlark.Dict
	parent *modelState
}

func redefineDict(pre starlark.StringDict) {
	olddict, pre["dict"] = pre["dict"].(*starlark.Builtin), starlark.NewBuiltin("dict", newModelstate)
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict
func newModelstate(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	d, err := starlark.Call(thread, olddict, args, kwargs)
	if err != nil {
		return nil, err
	}
	return &modelState{d: d.(*starlark.Dict)}, nil
}

func newModelState(size int, parent *modelState) *modelState {
	return &modelState{
		d:      starlark.NewDict(size),
		parent: parent,
	}
}

func (s *modelState) Clear() error { return s.d.Clear() }
func (s *modelState) Delete(k starlark.Value) (starlark.Value, bool, error) {
	if err := slValuePrintableASCII(k); err != nil {
		return nil, false, err
	}
	log.Printf("[NFO] Delete(%v)", k)
	return s.d.Delete(k)
}
func (s *modelState) Get(k starlark.Value) (starlark.Value, bool, error) {
	if err := slValuePrintableASCII(k); err != nil {
		return nil, false, err
	}
	log.Printf("[NFO] Get(%v) of %p", k, s)
	datapaths.current = append(datapaths.current, k)
	r, found, err := s.d.Get(k)
	// unreachable: SetLocal sets the thread-local value associated with the specified key. It must not be called after execution begins.
	// unfeasable: https://pkg.go.dev/go.starlark.net/syntax?tab=doc#Ident
	if found {
		datapath := fmt.Sprintf("%+v", datapaths.current)
		datapaths.ks = append(datapaths.ks, datapath)
		datapaths.vs = append(datapaths.vs, r)
		if len(datapaths.current) == 0 {
			panic(`racy datapaths`)
		}
		datapaths.current = datapaths.current[:len(datapaths.current)-1]
		if ss, ok := r.(*modelState); ok {
			log.Printf("[NFO] >>> statedictCallResponse:%p statedictModel:%p", statedictCallResponse, statedictModel)
			log.Printf("[NFO] >>> %p>%p|%p>%p %q {%p}[%v]: %s", s.parent, s, ss.parent, ss, datapath, s, k, r.String())
			for p := ss; p != nil; p = p.parent {
				pp := p.parent
				if pp == nil {
					break
				}
				var pstr string
				switch pp {
				case statedictCallResponse:
					pstr = "response"
				case statedictModel:
					pstr = "State"
				default:
					pstr = fmt.Sprintf("%p", pp)
				}
				log.Printf("[NFO] >>> p(%p): %s", p, pstr)
			}
		} else {
			log.Printf("[NFO] >>> %p>%p %q {%p}[%v]: %s", s.parent, s, datapath, s, k, r.String())
		}
	}
	return r, found, err
}
func (s *modelState) Items() []starlark.Tuple    { return s.d.Items() }
func (s *modelState) Keys() []starlark.Value     { return s.d.Keys() }
func (s *modelState) Len() int                   { return s.d.Len() }
func (s *modelState) Iterate() starlark.Iterator { return s.d.Iterate() }
func (s *modelState) SetKey(k, v starlark.Value) error {
	if err := slValuePrintableASCII(k); err != nil {
		return err
	}
	if err := slValueIsProtoable(v); err != nil {
		return err
	}
	//FIXME? visit & set parent
	if ss, ok := v.(*modelState); ok {
		ss.parent = s
		log.Printf("[NFO] %p>%p|%p>%p {%p}[%.50v] <- %.150v", s.parent, s, ss.parent, ss, s, k, v)
	} else {
		log.Printf("[NFO] %p>%p {%p}[%.50v] <- %.150v", s.parent, s, s, k, v)
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
