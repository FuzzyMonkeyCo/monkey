package runtime

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

const localCfg = "fuzzymonkey.star"

func init() {
	initExec()
}

// Runtime executes commands, resets and checks against the System Under Test
type Runtime struct {
	binTitle string

	thread     *starlark.Thread
	globals    starlark.StringDict
	modelState *modelState
	// EnvRead holds all the envs looked up on initial run
	envRead  map[string]string
	triggers []triggerActionAfterProbe

	models map[string]modeler.Interface

	eIds           []uint32
	shrinking      bool
	shrinkingTimes *uint32
	unshrunk       uint32

	tags map[string]string

	client *fm.ChBiDi

	logLevel             uint8
	ptype                string
	progress             progresser.Interface
	lastFuzzingProgress  *fm.Srv_FuzzingProgress
	testingCampaignStart time.Time
}

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name, ptype string, tags []string, vvv uint8) (rt *Runtime, err error) {
	if name == "" {
		err = errors.New("unnamed NewMonkey")
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
		ptype:    ptype,
		logLevel: vvv,
		models:   make(map[string]modeler.Interface, 1),
		globals:  make(starlark.StringDict, len(rt.builtins())+len(registeredModelers)),
	}
	log.Println("[NFO] registered modelers:", len(registeredModelers))
	for modelName, mdl := range registeredModelers {
		log.Printf("[DBG] registered modeler: %q", modelName)
		builtin := rt.modelMaker(modelName, mdl.NewFromKwargs)
		rt.globals[modelName] = starlark.NewBuiltin(modelName, builtin)
	}

	for t, b := range rt.builtins() {
		rt.globals[t] = starlark.NewBuiltin(t, b)
	}

	rt.thread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { as.ColorWRN.Println(msg) },
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, errors.New("load() disabled")
		},
	}
	rt.envRead = make(map[string]string)
	rt.triggers = make([]triggerActionAfterProbe, 0)

	log.Println("[NFO] loading starlark config from", localCfg)
	start := time.Now()
	if err = rt.loadCfg(localCfg); err != nil {
		return
	}
	log.Println("[NFO] loaded", localCfg, "in", time.Since(start))

	for _, kv := range tags {
		if idx := strings.IndexAny(kv, "="); idx != -1 {
			k, v := kv[:idx], kv[idx+1:]
			// NOTE: validation also client side for shorter dev loop
			if err = printableASCII(k); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if v == "" {
				err = fmt.Errorf("value for tag %q is empty", k)
				log.Println("[ERR]", err)
				return
			}
			rt.tags[k] = v
		} else {
			err = fmt.Errorf("tags must follow key=value format: %q", kv)
			log.Println("[ERR]", err)
			return
		}
	}

	return
}

func (rt *Runtime) loadCfg(localCfg string) (err error) {
	log.Printf("[DBG] starlark globals: %d", len(rt.globals))
	for k, v := range rt.globals {
		log.Printf("[DBG] starlark global %q: %+v", k, v)
	}
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

	log.Printf("[DBG] models defined: %d", len(rt.models))
	for k, v := range rt.models {
		log.Printf("[DBG] defined model %q: %+v", k, v)
	}
	if len(rt.models) == 0 {
		err = errors.New("no models registered")
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[NFO] frozen envs: %d", len(rt.envRead))
	for k, v := range rt.envRead {
		log.Printf("[NFO] env frozen %q: %+v", k, v)
	}
	log.Printf("[NFO] readying %d triggers", len(rt.triggers))

	for t := range rt.builtins() {
		delete(rt.globals, t)
	}
	log.Printf("[DBG] starlark globals: %d", len(rt.globals))
	for k, v := range rt.globals {
		log.Printf("[DBG] starlark global %q: %+v", k, v)
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
	log.Printf("[DBG] starlark globals: %d", len(rt.globals))
	for k, v := range rt.globals {
		log.Printf("[DBG] starlark global %q: %+v", k, v)
	}
	return
}
