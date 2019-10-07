package pkg

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"

	"go.starlark.net/starlark"
)

// NewMonkey parses and optionally pretty-prints configuration
func NewMonkey(name string, showCfg bool) (mnk *fm.monkey, err error) {
	binTitle = name
	const localCfg = "fuzzymonkey.star"
	if _, err = os.Stat(localCfg); os.IsNotExist(err) {
		log.Println("[ERR]", err)
		ColorERR.Printf("You must provide a readable %q file in the current directory.\n", localCfg)
		return
	}

	mnk = &monkey{}
	mnk.globals = make(starlark.StringDict, 2+len(registeredIRModels))
	for modelName, modeler := range registeredIRModels {
		if _, ok := UserCfg_Kind_value[modelName]; !ok {
			err = fmt.Errorf("unexpected model kind: %q", modelName)
			return
		}
		builtin := mnk.modelMaker(modelName, modeler)
		mnk.globals[modelName] = starlark.NewBuiltin(modelName, builtin)
	}
	mnk.globals[tEnv] = starlark.NewBuiltin(tEnv, mnk.bEnv)
	mnk.globals[tTriggerActionAfterProbe] = starlark.NewBuiltin(tTriggerActionAfterProbe, mnk.bTriggerActionAfterProbe)
	mnk.thread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { ColorWRN.Println(msg) },
	}
	mnk.envRead = make(map[string]string)
	mnk.triggers = make([]triggerActionAfterProbe, 0)

	start := time.Now()
	if err = mnk.loadCfg(localCfg, showCfg); err != nil {
		return
	}
	log.Println("[NFO] loaded", localCfg, "in", time.Since(start))

	mnk.usage = os.Args
	return
}
