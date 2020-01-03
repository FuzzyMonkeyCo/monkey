package runtime

import (
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	modeler_openapiv3 "github.com/FuzzyMonkeyCo/monkey/pkg/modeler/openapiv3"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

var registeredModelers = map[string]modeler.Interface{
	"OpenAPIv3": (*modeler_openapiv3.T)(nil),
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

			reserved := false
			if err = printableASCII(key); err != nil {
				err = errors.Wrap(err, "illegal field")
				log.Println("[ERR]", err)
				return
			}
			for i, c := range key {
				if i == 0 && unicode.IsUpper(c) {
					reserved = true
					break
				}
			}
			if !reserved {
				u[key] = v
			} else {
				r[key] = v
			}
		}

		if modelName == "" {
			err = errors.New("model's Name must be set")
			log.Println("[ERR]", err)
			return
		}
		if err = printableASCII(modelerName); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if modelName != strings.ToLower(modelName) {
			err = fmt.Errorf("model name %q must be lowercase", modelName)
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
