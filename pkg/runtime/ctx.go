package runtime

import (
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func newCtx(state, request, response starlark.Value) starlark.Value {
	return &starlarkstruct.Module{
		Name: "ctx",
		Members: starlark.StringDict{
			"request":  request,
			"response": response,
			"state":    state,
		},
	}
}

// func (m *starlarkCtx) String() string                           { return "ctx" }
// func (m *starlarkCtx) Type() string                             { return "ctx" }
// func (m *starlarkCtx) Freeze()                                  {}
// func (m *starlarkCtx) Truth() starlark.Bool                     { return false }
// func (m *starlarkCtx) Hash() (uint32, error)                    { return 0, errors.New("unhashable type: ctx") }
// func (m *starlarkCtx) AttrNames() []string                      { return ctxAttrNames }
// func (m *starlarkCtx) Attr(name string) (starlark.Value, error) { return m.attrFor(name) }

// func (m *starlarkCtx) attrFor(name string) (starlark.Value, error) {
// 	b := ctxAttrs[name]
// 	if b == nil {
// 		return nil, nil // no such method
// 	}
// 	return b.BindReceiver(m), nil
// }

// var (
// 	ctxAttrs = map[string]*starlark.Builtin{
// 		"request": starlark.NewBuiltin("request", func(
// 			thread *starlark.Thread,
// 			b *starlark.Builtin,
// 			args starlark.Tuple,
// 			kwargs []starlark.Tuple,
// 		) (starlark.Value, error) {
// 			log.Println("[ERR]", b.Receiver())
// 			if len(args) != 0 || len(kwargs) != 0 {
// 				return nil, fmt.Errorf("unexpected [kw]args: %+v %+v", args, kwargs)
// 			}
// 			m, ok := b.Receiver().(*starlarkCtx) //FIXME: get m without typeassert
// 			if !ok {
// 				return nil, fmt.Errorf("unreachable: %T %+v", b.Receiver(), b.Receiver())
// 			}
// 			return m.req, nil
// 		}),

// 		"response": starlark.NewBuiltin("response", func(
// 			thread *starlark.Thread,
// 			b *starlark.Builtin,
// 			args starlark.Tuple,
// 			kwargs []starlark.Tuple,
// 		) (starlark.Value, error) {
// 			log.Println("[ERR]", b.Receiver())
// 			if len(args) != 0 || len(kwargs) != 0 {
// 				return nil, fmt.Errorf("unexpected [kw]args: %+v %+v", args, kwargs)
// 			}
// 			m, ok := b.Receiver().(*starlarkCtx) //FIXME: get m without typeassert
// 			if !ok {
// 				return nil, fmt.Errorf("unreachable: %T %+v", b.Receiver(), b.Receiver())
// 			}
// 			return m.rep, nil
// 		}),

// 		"state": starlark.NewBuiltin("state", func(
// 			thread *starlark.Thread,
// 			b *starlark.Builtin,
// 			args starlark.Tuple,
// 			kwargs []starlark.Tuple,
// 		) (starlark.Value, error) {
// 			log.Println("[ERR]", b.Receiver())
// 			if len(args) != 0 || len(kwargs) != 0 {
// 				return nil, fmt.Errorf("unexpected [kw]args: %+v %+v", args, kwargs)
// 			}
// 			m, ok := b.Receiver().(*starlarkCtx) //FIXME: get m without typeassert
// 			if !ok {
// 				return nil, fmt.Errorf("unreachable: %T %+v", b.Receiver(), b.Receiver())
// 			}
// 			return m.state, nil
// 		}),
// 	}

// 	ctxAttrNames = func() []string {
// 		names := make([]string, 0, len(ctxAttrs))
// 		for name := range ctxAttrs {
// 			names = append(names, name)
// 		}
// 		sort.Strings(names)
// 		return names
// 	}()
// )
