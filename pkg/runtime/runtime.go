package runtime

import (
	"fmt"
	"log"
	"os"
	"time"
	"unicode"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

var binTitle string

// Bintitle TODO
func Bintitle() string {
	return binTitle
}

type runtime struct {
	eIds     []uint32
	Ntensity uint32

	thread     *starlark.Thread
	globals    starlark.StringDict
	modelState *modelState
	// EnvRead holds all the envs looked up on initial run
	envRead  map[string]string
	triggers []triggerActionAfterProbe

	models []modeler.Interface

	client fm.Client

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
	rt.globals = make(starlark.StringDict, len(rt.builtins())+len(registeredIRModels))
	as.ColorERR.Printf(">>> registeredIRModels: %+v\n", registeredIRModels)
	for modelName, mdlr := range registeredIRModels {
		if _, ok := fm.Clt_Msg_Fuzz_ModelKind_value[modelName]; !ok {
			err = fmt.Errorf("unexpected model kind: %q", modelName)
			return
		}
		builtin := rt.modelMaker(modelName, mdlr)
		rt.globals[modelName] = starlark.NewBuiltin(modelName, builtin)
	}

	for t, b := range rt.builtins() {
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

func (rt *runtime) loadCfg(localCfg string) (err error) {
	as.ColorERR.Printf(">>> globals: %+v\n", rt.globals)
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
	as.ColorERR.Printf(">>> modelers: %v\n", rt.models)
	if len(rt.models) == 0 {
		err = errors.New("no modelers are registered")
		log.Println("[ERR]", err)
		return
	}

	as.ColorERR.Printf(">>> envs: %+v\n", rt.envRead)
	as.ColorERR.Printf(">>> trigs: %+v\n", rt.triggers)

	for t := range rt.builtins() {
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
