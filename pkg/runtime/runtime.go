package runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"unicode"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/do/fuzz/reset"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

var rtBuiltins = map[string]rtBuiltin{
	"Env":                     rt.bEnv,
	"TriggerActionAfterProbe": rt.bTriggerActionAfterProbe,
}

type runtime struct {
	EIDs     []uint32
	Ntensity uint32

	thread     *starlark.Thread
	globals    starlark.StringDict
	modelState *modelState
	// EnvRead holds all the envs looked up on initial run
	envRead  map[string]string
	triggers []triggerActionAfterProbe

	modelers []modeler.Modeler

	client *fm.Client

	progress ui.Progresser
}

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name string) (rt *runtime, err error) {
	binTitle = name
	const localCfg = "fuzzymonkey.star"
	if _, err = os.Stat(localCfg); os.IsNotExist(err) {
		log.Println("[ERR]", err)
		as.ColorERR.Printf("You must provide a readable %q file in the current directory.\n", localCfg)
		return
	}

	rt = &runtime{}
	rt.globals = make(starlark.StringDict, 2+len(registeredIRModels))
	for modelName, modeler := range registeredIRModels {
		if _, ok := fm.Clt_Msg_Fuzz_ModelKind_value[modelName]; !ok {
			err = fmt.Errorf("unexpected model kind: %q", modelName)
			return
		}
		builtin := rt.modelMaker(modelName, modeler)
		rt.globals[modelName] = starlark.NewBuiltin(modelName, builtin)
	}

	for t, b := range rtBuiltins {
		rt.globals[t] = starlark.NewBuiltin(t, b)
	}

	rt.thread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { as.ColorWRN.Println(msg) },
	}
	rt.envRead = make(map[string]string)
	rt.triggers = make([]triggerActionAfterProbe, 0)

	start := time.Now()
	if err = rt.loadCfg(localCfg); err != nil {
		return
	}
	log.Println("[NFO] loaded", localCfg, "in", time.Since(start))

	return
}

type rtBuiltin func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

func (rt *runtime) modelMaker(modelName string, mdlr ModelerFunc) rtBuiltin {
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
		mo, err := mdlr(u)
		if err != nil {
			if modelerErr, ok := err.(*ModelerError); ok {
				modelerErr.SetModelerName(modelName)
				err = modelerErr
			}
			log.Println("[ERR]", err)
			return
		}
		var resetter reset.SUTResetter
		if resetter, err = reset.NewFromKwargs(fname, r); err != nil {
			return
		}
		mo.SetSUTResetter(resetter)

		rt.modelers = append(rt.modelers, mo)
		return
	}
}

func (rt *runtime) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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
	rt.envRead[envStr] = read
	return starlark.String(read), nil
}

type triggerActionAfterProbe struct {
	Name              starlark.String
	Probe             starlark.Tuple
	Predicate, Action *starlark.Function
}

func (rt *runtime) bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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
	rt.triggers = append(rt.triggers, trigger)
	return starlark.None, nil
}

func (rt *runtime) loadCfg(localCfg string) (err error) {
	if rt.globals, err = starlark.ExecFile(rt.thread, localCfg, nil, rt.globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	// Ensure at least one model was defined
	as.ColorERR.Printf(">>> modelers: %v\n", rt.modelers)
	if len(rt.modelers) == 0 {
		err = errors.New("no modelers are registered")
		log.Println("[ERR]", err)
		return
	}

	as.ColorERR.Printf(">>> envs: %+v\n", rt.envRead)
	as.ColorERR.Printf(">>> trigs: %+v\n", rt.triggers)

	for _, t := range rtBuiltins {
		delete(rt.globals, t)
	}

	const tState = "State"
	if state, ok := rt.globals[tState]; ok {
		d, ok := state.(*starlark.Dict)
		if !ok {
			err = fmt.Errorf("monkey State must be a dict, got: %s", state.Type())
			log.Println("[ERR]", err)
			return
		}
		delete(rt.globals, tState)
		rt.modelState = newModelState(d.Len())
		for _, kd := range d.Items() {
			k, v := kd.Index(0), kd.Index(1)
			// Ensure State keys are all String.s
			if err = slValuePrintableASCII(k); err != nil {
				err = errors.Wrap(err, "illegal State key")
				log.Println("[ERR]", err)
				return
			}
			// Ensure State values are all literals
			switch v.(type) {
			case starlark.NoneType, starlark.Bool:
			case starlark.Int, starlark.Float:
			case starlark.String:
			case *starlark.List, *starlark.Dict, *starlark.Set:
			default:
				err = fmt.Errorf("all initial State values must be litterals: State[%s] is %s", k.String(), v.Type())
				log.Println("[ERR]", err)
				return
			}
			as.ColorERR.Printf(">>> modelState: SetKey(%v, %v)\n", k, v)
			var vv starlark.Value
			if vv, err = slValueCopy(v); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if err = rt.modelState.SetKey(k, vv); err != nil {
				log.Println("[ERR]", err)
				return
			}
		}
	} else {
		rt.modelState = newModelState(0)
	}
	for key := range rt.globals {
		if err = printableASCII(key); err != nil {
			err = errors.Wrap(err, "illegal export")
			log.Println("[ERR]", err)
			return
		}
		for _, c := range key {
			if unicode.IsUpper(c) {
				err = fmt.Errorf("user defined exports must not be uppercase: %q", key)
				log.Println("[ERR]", err)
				return
			}
			break
		}
	}
	log.Println("[NFO] starlark cfg globals:", len(rt.globals.Keys()))
	as.ColorERR.Printf(">>> globals: %#v\n", rt.globals)
	return
}
