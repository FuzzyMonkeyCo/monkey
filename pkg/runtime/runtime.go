package runtime

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	"go.starlark.net/starlark"
)

const localCfg = "fuzzymonkey.star"

func init() {
	initExec()
}

// Runtime executes commands, resets and checks against the System Under Test
type Runtime struct {
	binTitle string

	thread  *starlark.Thread
	globals starlark.StringDict

	envRead map[string]string // holds all the envs looked up on initial run
	models  map[string]modeler.Interface
	files   map[string]string
	checks  map[string]*check

	client    *fm.ChBiDi
	eIds      []uint32
	labels    map[string]string
	cleanedup bool

	progress            progresser.Interface
	lastFuzzingProgress *fm.Srv_FuzzingProgress
	fuzzingStartedAt    time.Time
}

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name string, labels []string) (rt *Runtime, err error) {
	if name == "" {
		err = errors.New("unnamed NewMonkey")
		log.Println("[ERR]", err)
		return
	}

	var localCfgContents []byte
	if localCfgContents, err = ioutil.ReadFile(localCfg); err != nil {
		log.Println("[ERR]", err)
		as.ColorERR.Printf("You must provide a readable %q file in the current directory.\n", localCfg)
		return
	}

	rt = &Runtime{
		binTitle: name,
		files:    map[string]string{localCfg: string(localCfgContents)},
		models:   make(map[string]modeler.Interface, 1),
		globals:  make(starlark.StringDict, len(rt.builtins())+len(registeredModelers)),
		thread: &starlark.Thread{
			Name:  "cfg",
			Load:  loadDisabled,
			Print: func(_ *starlark.Thread, msg string) { as.ColorWRN.Println(msg) },
		},
		envRead: make(map[string]string),
		checks:  make(map[string]*check),
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

	log.Println("[NFO] loading starlark config from", localCfg)
	start := time.Now()
	if err = rt.loadCfg(localCfg); err != nil {
		return
	}
	log.Println("[NFO] loaded", localCfg, "in", time.Since(start))

	for _, kv := range labels {
		if idx := strings.IndexAny(kv, "="); idx != -1 {
			k, v := kv[:idx], kv[idx+1:]
			// NOTE: validation also client side for shorter dev loop
			if err = printableASCII(k); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if v == "" {
				err = fmt.Errorf("value for label %q is empty", k)
				log.Println("[ERR]", err)
				return
			}
			rt.labels[k] = v
		} else {
			err = fmt.Errorf("labels must follow key=value format: %q", kv)
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

	log.Printf("[NFO] checks defined: %d", len(rt.checks))
	for k, v := range rt.checks {
		log.Printf("[NFO] defined check %q: %+v", k, v)
	}

	for t := range rt.builtins() {
		delete(rt.globals, t)
	}
	for key := range rt.globals {
		if err = printableASCII(key); err != nil {
			err = fmt.Errorf("illegal export: %v", err)
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
