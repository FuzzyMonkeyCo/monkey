package pkg

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"go.starlark.net/starlark"
)

// TODO: turn these into methods of *runtime
const (
	tEnv                     = "Env"
	tState                   = "State"
	tTriggerActionAfterProbe = "TriggerActionAfterProbe"
)

type runtime struct {
	thread     *starlark.Thread
	globals    starlark.StringDict
	modelState *modelState
	// EnvRead holds all the envs looked up on initial run
	envRead  map[string]string
	triggers []triggerActionAfterProbe
	modelers []modeler.Modeler
}

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name string, showCfg bool) (rt *runtime, err error) {
	binTitle = name
	const localCfg = "fuzzymonkey.star"
	if _, err = os.Stat(localCfg); os.IsNotExist(err) {
		log.Println("[ERR]", err)
		ColorERR.Printf("You must provide a readable %q file in the current directory.\n", localCfg)
		return
	}

	rt = &runtime{}
	rt.globals = make(starlark.StringDict, 2+len(registeredIRModels))
	for modelName, modeler := range registeredIRModels {
		if _, ok := UserCfg_Kind_value[modelName]; !ok {
			err = fmt.Errorf("unexpected model kind: %q", modelName)
			return
		}
		builtin := rt.modelMaker(modelName, modeler)
		rt.globals[modelName] = starlark.NewBuiltin(modelName, builtin)
	}
	rt.globals[tEnv] = starlark.NewBuiltin(tEnv, rt.bEnv)
	rt.globals[tTriggerActionAfterProbe] = starlark.NewBuiltin(tTriggerActionAfterProbe, rt.bTriggerActionAfterProbe)
	rt.thread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { ColorWRN.Println(msg) },
	}
	rt.envRead = make(map[string]string)
	rt.triggers = make([]triggerActionAfterProbe, 0)

	start := time.Now()
	if err = rt.loadCfg(localCfg, showCfg); err != nil {
		return
	}
	log.Println("[NFO] loaded", localCfg, "in", time.Since(start))

	// FIXME: mnk.usage = os.Args
	return
}

type slBuiltin func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

func (rt *runtime) modelMaker(modelName string, mdlr modeler.Func) slBuiltin {
	return func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (ret starlark.Value, err error) {
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
			err = modelerErr.Error(modelName)
			log.Println("[ERR]", err)
			return
		}
		var resetter reset.SUTResetter
		if resetter, err = reset.NewFromKwargs(fname, r); err != nil {
			return
		}
		mo.SetSUTResetter(resetter)

		mnkrt.modelers = append(rt.modelers, mo)
		return
	}
}

func (mnk *monkey) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()
	// FIXME: actually maybe read env from Exec shell? These shells should inherit user env anyway?
	read, ok := os.LookupEnv(envStr)
	if !ok {
		return def, nil
	}
	mnk.envRead[envStr] = read
	return starlark.String(read), nil
}

type triggerActionAfterProbe struct {
	Name              starlark.String
	Probe             starlark.Tuple
	Predicate, Action *starlark.Function
}

func (mnk *monkey) bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var trigger triggerActionAfterProbe
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"name?", &trigger.Name,
		"probe", &trigger.Probe,
		"predicate", &trigger.Predicate,
		"action", &trigger.Action,
	); err != nil {
		return nil, err
	}
	// TODO: enforce arities
	log.Println("[NFO] registering", b.Name(), trigger)
	if name := trigger.Name.GoString(); name == "" {
		trigger.Name = starlark.String(trigger.Action.Name())
		// TODO: complain if trigger.Name == "lambda"
	}
	mnk.triggers = append(mnk.triggers, trigger)
	return starlark.None, nil
}
