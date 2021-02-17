package runtime

import (
	"errors"
	"fmt"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler/openapiv3"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"go.starlark.net/starlark"
)

var registeredModelers = map[string]modeler.Interface{
	"OpenAPIv3": (*openapiv3.T)(nil),
}

func (rt *Runtime) modelMaker(modelerName string, mdlr modeler.Func) builtin {
	return func(
		th *starlark.Thread,
		b *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (ret starlark.Value, err error) {
		ret = starlark.None
		fname := b.Name()
		if args.Len() != 0 {
			err = fmt.Errorf("%s(...) does not take positional arguments", fname)
			log.Println("[ERR]", err)
			return
		}

		var modelName string
		u := make(starlark.StringDict, len(kwargs))
		r := make(starlark.StringDict, len(kwargs))
		for _, kv := range kwargs {
			k, v := kv.Index(0), kv.Index(1)
			key := k.(starlark.String).GoString()

			if field := "name"; field == key {
				if name, ok := v.(starlark.String); ok {
					modelName = name.GoString()
					continue
				} else {
					modelerErr := modeler.NewError(field, "a string", v.Type())
					modelerErr.SetModelerName(modelerName)
					err = modelerErr
					log.Println("[ERR]", err)
					return
				}
			}

			var reserved bool
			if reserved, err = printableASCIItmp(key); err != nil { // FIXME: no more CamelReserved kwargs
				err = fmt.Errorf("illegal field: %v", err)
				log.Println("[ERR]", err)
				return
			}
			if !reserved {
				u[key] = v
			} else {
				r[key] = v
			}
		}

		if modelName == "" {
			err = errors.New("model's name must be set")
			log.Println("[ERR]", err)
			return
		}
		if err = tags.LegalName(modelName); err != nil {
			log.Println("[ERR]", err)
			return
		}

		model, modelerErr := mdlr(u)
		if modelerErr != nil {
			modelerErr.SetModelerName(modelerName)
			err = modelerErr
			log.Println("[ERR]", err)
			return
		}

		// FIXME: actually only couple resetter & modeler through model name
		var rsttr resetter.Interface
		if rsttr, err = newFromKwargs(fname, r); err != nil {
			return
		}
		model.SetResetter(rsttr)

		if _, ok := rt.models[modelName]; ok {
			err = fmt.Errorf("model name %q is already defined", modelName)
			log.Println("[ERR]", err)
			return
		}
		rt.models[modelName] = model
		return
	}
}
