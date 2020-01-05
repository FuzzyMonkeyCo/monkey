package house

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

const localCfg = "fuzzymonkey.star"

type Runtime struct {
	binTitle string

	eIds []uint32

	thread     *starlark.Thread
	globals    starlark.StringDict
	modelState *modelState
	// EnvRead holds all the envs looked up on initial run
	envRead  map[string]string
	triggers []triggerActionAfterProbe

	models map[string]modeler.Interface

	client fm.FuzzyMonkey_DoClient

	progress ui.Progresser
}

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name string) (rt *Runtime, err error) {
	if name == "" {
		err = errors.New("Ook!")
		log.Println("[ERR]", err)
		return
	}

	if _, err = os.Stat(localCfg); os.IsNotExist(err) {
		log.Println("[ERR]", err)
		as.ColorERR.Printf("You must provide a readable %q file in the current directory.\n", localCfg)
		return
	}

	rt = &Runtime{
		binTitle: name,
		models:   make(map[string]modeler.Interface, 1),
		globals:  make(starlark.StringDict, len(rt.builtins())+len(registeredModelers)),
	}
	log.Printf("[NFO] %d registered modeler(s):", len(registeredModelers))
	for modelName, mdl := range registeredModelers {
		log.Printf("[DBG] registered modeler: %q", modelName)
		if _, ok := fm.Clt_Fuzz_ModelKind_value[modelName]; !ok {
			err = fmt.Errorf("unexpected model kind: %q", modelName)
			return
		}
		builtin := rt.modelMaker(modelName, mdl.NewFromKwargs)
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

func (rt *Runtime) loadCfg(localCfg string) (err error) {
	log.Printf("[DBG] %d starlark globals: %+v", len(rt.globals), rt.globals)
	if rt.globals, err = starlark.ExecFile(rt.thread, localCfg, nil, rt.globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			as.ColorWRN.Println(bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[DBG] %d defined models: %v", len(rt.models), rt.models)
	if len(rt.models) == 0 {
		err = errors.New("no models registered")
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[NFO] froze %d envs: %+v", len(rt.envRead), rt.envRead)
	log.Printf("[NFO] readying %d triggers", len(rt.triggers))

	for t := range rt.builtins() {
		delete(rt.globals, t)
	}
	log.Printf("[DBG] %d starlark globals: %+v", len(rt.globals), rt.globals)

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
	log.Printf("[DBG] %d starlark globals: %+v", len(rt.globals), rt.globals)
	return
}
