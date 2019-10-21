package runtime

import (
	"fmt"
	"log"
	"unicode"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

var registeredIRModels = make(map[string]newModelerFunc)

type newModelerFunc func(starlark.StringDict) (modeler.Interface, *modeler.Error)

// RegisterModeler TODO
func RegisterModeler(name string, mdlr modeler.Interface) {
	if _, ok := registeredIRModels[name]; ok {
		panic(fmt.Sprintf("modeler %q is already registered", name))
	}
	registeredIRModels[name] = mdlr.NewFromKwargs
}

func (rt *runtime) modelMaker(modelName string, mdlr newModelerFunc) builtin {
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
			return
		}

		u := make(starlark.StringDict, len(kwargs))
		r := make(starlark.StringDict, len(kwargs))
		for _, kv := range kwargs {
			k, v := kv.Index(0), kv.Index(1)
			key := k.(starlark.String).GoString()
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

		mo, modelerErr := mdlr(u)
		if modelerErr != nil {
			modelerErr.SetModelerName(modelName)
			err = modelerErr
			log.Println("[ERR]", err)
			return
		}

		var rsttr resetter.Interface
		if rsttr, err = newFromKwargs(fname, r); err != nil {
			return
		}
		mo.SetResetter(rsttr)

		rt.models = append(rt.models, mo)
		return
	}
}
