package runtime

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
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
	files   map[string]string

	models      map[string]modeler.Interface
	modelsNames []string

	resetters      map[string]resetter.Interface // FIXME: rename: inits initers zeroers
	resettersNames []string

	checks      map[string]*check
	checksNames []string

	client       *fm.ChBiDi
	selectedEIDs map[string]*fm.Uint32S
	labels       map[string]string
	cleanedup    bool

	progress            progresser.Interface
	lastFuzzingProgress *fm.Srv_FuzzingProgress
	fuzzingStartedAt    time.Time
}

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name string, arglabels []string) (rt *Runtime, err error) {
	if name == "" {
		err = errors.New("unnamed NewMonkey")
		log.Println("[ERR]", err)
		return
	}

	labels := make(map[string]string, len(arglabels))
	for _, kv := range arglabels {
		if idx := strings.IndexAny(kv, "="); idx != -1 {
			k, v := kv[:idx], kv[idx+1:]
			if err = tags.LegalName(k); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if v == "" {
				err = fmt.Errorf("value for label %q is empty", k)
				log.Println("[ERR]", err)
				return
			}
			labels[k] = v
		} else {
			err = fmt.Errorf("labels must follow key=value format: %q", kv)
			log.Println("[ERR]", err)
			return
		}
	}

	var localCfgContents []byte
	if localCfgContents, err = localcfgdata(); err != nil {
		log.Println("[ERR]", err)
		as.ColorERR.Printf("You must provide a readable %q file in the current directory.\n", localCfg)
		return
	}

	r := &Runtime{
		binTitle:  name,
		files:     map[string]string{localCfg: string(localCfgContents)},
		models:    make(map[string]modeler.Interface, moduleModelers),
		resetters: make(map[string]resetter.Interface, moduleResetters),
		thread: &starlark.Thread{
			Name:  "cfg",
			Load:  loadDisabled,
			Print: func(_ *starlark.Thread, msg string) { as.ColorWRN.Println(msg) },
		},
		labels:  labels,
		envRead: make(map[string]string),
		checks:  make(map[string]*check),
	}
	r.globals = starlark.StringDict{"monkey": r.newModule()}

	log.Println("[NFO] loading starlark config from", localCfg)
	start := time.Now()
	if err = r.loadCfg(); err != nil {
		return
	}
	log.Println("[NFO] loaded", localCfg, "in", time.Since(start))

	rt = r
	return
}

func (rt *Runtime) loadCfg() (err error) {
	log.Printf("[DBG] starlark globals: %d", len(rt.globals))
	for k, v := range rt.globals {
		log.Printf("[DBG] starlark global %q: %+v", k, v)
	}

	if rt.globals, err = starlark.ExecFile(rt.thread, localCfg, rt.files[localCfg], rt.globals); err != nil {
		log.Println("[ERR]", err)
		err = starTrickError(err)
		return
	}
	if err = starlarktruth.Close(rt.thread); err != nil {
		log.Println("[ERR]", err)
		// Incomplete `assert that()` call
		err = starTrickError(err)
		return
	}

	log.Printf("[DBG] models defined: %d", len(rt.models))
	_ = rt.forEachModel(func(name string, mdl modeler.Interface) error {
		log.Printf("[DBG] defined model %q: %+v", name, mdl)
		return nil
	})
	if len(rt.models) == 0 {
		err = errors.New("no models registered")
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[DBG] resetters defined: %d", len(rt.resetters))
	if err = rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		log.Printf("[DBG] defined resetter %q: %+v", name, rsttr)
		for _, mdl := range rsttr.Provides() {
			if _, ok := rt.models[mdl]; !ok {
				err := fmt.Errorf("resetter %q provides undefined %q", name, mdl)
				log.Println("[ERR]", err)
				return err
			}
		}
		return nil
	}); err != nil {
		return
	}

	log.Printf("[NFO] frozen envs: %d", len(rt.envRead))
	for name, value := range rt.envRead {
		log.Printf("[NFO] froze env %q: %+v", name, value)
	}

	log.Printf("[NFO] checks defined: %d", len(rt.checks))
	_ = rt.forEachCheck(func(name string, chk *check) error {
		log.Printf("[NFO] defined check %q: %+v", name, chk)
		return nil
	})

	delete(rt.globals, "monkey")
	log.Printf("[DBG] starlark globals: %d", len(rt.globals))
	for name, value := range rt.globals {
		if err = tags.LegalName(name); err != nil {
			err = fmt.Errorf("illegal name %q: %v", name, err)
			log.Println("[ERR]", err)
			return
		}
		log.Printf("[DBG] starlark global %q: %+v", name, value)
	}
	return
}
