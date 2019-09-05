package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"go.starlark.net/starlark"
	yaml "gopkg.in/yaml.v2"
)

const (
	// LocalCfg is the path of Monkey's config file
	LocalCfg = ".fuzzymonkey.yml"

	lastCfgVersion = 1
	defaultCfgHost = "http://localhost:3000"
)

// FIXME: use new Modeler intf struc to pass these
var addHeaderAuthorization, addHost *string

// NewCfg parses Monkey configuration, optionally pretty-printing it
func NewCfg(showCfg bool) (cfg *UserCfg, err error) {
	fd, err := os.Open(LocalCfg)
	if err != nil {
		log.Println("[ERR]", err)
		errFmt := "You must provide a readable %s file in the current directory.\n"
		ColorERR.Printf(errFmt, LocalCfg)
		return
	}
	defer fd.Close()

	var config []byte
	if config, err = ioutil.ReadAll(fd); err != nil {
		log.Println("[ERR]", err)
		return
	}

	start := time.Now()
	if _, err = loadCfg(config, showCfg); err != nil {
		return
	}
	log.Println(">>>", time.Since(start))

	if cfg, err = parseCfg(config, showCfg); err == nil {
		cfg.Usage = os.Args
	}
	return
}

// Spec describes a spec
type Spec struct {
	Version   int
	Model     Modeler
	Overrides map[string]string
}

// Modeler describes any model
type Modeler interface {
	Kind() ModelKind
	pp()
}

type ModelerFunc func(d starlark.StringDict) (Modeler, error)
type slBuiltin func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

var registeredIRModels = map[string]ModelerFunc{
	"OpenAPIv3": func(d starlark.StringDict) (Modeler, error) {
		mo := ModelOpenAPIv3{}
		var (
			found      bool
			file, host starlark.Value
		)

		if file, found = d["file"]; !found || file.Type() != "string" {
			return nil, errors.New("OpenAPIv3(file = ...) must be a string")
		}
		mo.File = file.(starlark.String).GoString()

		if host, found = d["host"]; found && host.Type() != "string" {
			return nil, errors.New("OpenAPIv3(host = ...) must be a string")
		}
		h := host.(starlark.String).GoString()
		addHost = &h

		return mo, nil
	},
}

func modelMaker(modeler ModelerFunc) slBuiltin {
	return func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		fname := b.Name()

		if args.Len() != 0 {
			return nil, fmt.Errorf("%s(...) does not take positional arguments", fname)
		}

		d := make(starlark.StringDict, len(kwargs))
		for _, kv := range kwargs {
			k, v := kv.Index(0), kv.Index(1)
			key := k.(starlark.String).GoString()
			// FIXME: prempt Exec* fields + other Capitalized keys
			d[key] = v
		}
		mo, err := modeler(d)
		if err != nil {
			return nil, err
		}

		userRTLang.Models = append(userRTLang.Models, mo)
		return starlark.None, nil
	}
}

// def OpenAPIv3(**kwargs):
// # AddNewModel takes care of popping exec_* and setting ModelKind.
// 	model = AddNewModel(kwargs, ` + strconv.Itoa(int(ModelKindOpenAPIv3)) + `)
//  aaaah let's just do a Go model constructor for now...
// 		'file': kwargs.pop('file'),
// # This should be ensured by the framework:
// 	if len(kwargs) != 0:
// 		fail("Unexpected arguments to <...>():", kwargs)
// 	return model

// ModelKind enumerates model kinds
type ModelKind int

const (
	// ModelKindUnset is the empty ModelKind
	ModelKindUnset ModelKind = iota
	// ModelKindOpenAPIv3 is for OpenAPIv3 models
	ModelKindOpenAPIv3
)

// ModelOpenAPIv3 describes OpenAPIv3 models
type ModelOpenAPIv3 struct {
	File string
}

// Kind returns a ModelKind
func (m ModelOpenAPIv3) Kind() ModelKind { return ModelKindOpenAPIv3 }
func (m ModelOpenAPIv3) pp()             { fmt.Println(m) }

// SUT describes ways to reset the system under test to a known initial state
type SUT struct {
	Start, Reset, Stop []string
}

var userRTLang struct {
	Thread     *starlark.Thread
	Globals    starlark.StringDict
	ModelState *starlark.Dict
	// InitialRun is true only when localcfg is first interpreted
	InitialRun bool
	// EnvRead holds all the envs looked up while InitialRun is true
	EnvRead  map[string]string
	Triggers []triggerActionAfterProbe

	Models []Modeler
}

// TODO: turn these into methods of userRTLang
const (
	tState                   = "State"
	tEnv                     = "Env"
	tTriggerActionAfterProbe = "TriggerActionAfterProbe"
)

func bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()
	if !userRTLang.InitialRun {
		// FIXME: just don't declare bEnv during InitialRun
		return nil, fmt.Errorf("calling %s(%q) is forbidden", b.Name(), envStr)
	}

	// FIXME: actually maybe read env from Exec shell? These shells should inherit user env anyway?
	read, ok := os.LookupEnv(envStr)
	if !ok {
		return def, nil
	}
	userRTLang.EnvRead[envStr] = read
	return starlark.String(read), nil
}

type triggerActionAfterProbe struct {
	Probe             starlark.Tuple
	Predicate, Action *starlark.Function
}

func bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var trigger triggerActionAfterProbe
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"probe", &trigger.Probe,
		"predicate", &trigger.Predicate,
		"action", &trigger.Action,
	); err != nil {
		return nil, err
	}
	log.Println("[NFO] registering", b.Name(), trigger)
	userRTLang.Triggers = append(userRTLang.Triggers, trigger)
	return starlark.None, nil
}

func loadCfg(config []byte, showCfg bool) (globals starlark.StringDict, err error) {
	const localCfg = "fuzzymonkey.star"

	userRTLang.Globals = make(starlark.StringDict, 2+len(registeredIRModels))
	for modelName, modeler := range registeredIRModels {
		if _, ok := UserCfg_Kind_value[modelName]; !ok {
			return nil, fmt.Errorf("unexpected model kind: %q", modelName)
		}
		userRTLang.Globals[modelName] = starlark.NewBuiltin(modelName, modelMaker(modeler))
	}

	// valSUT := SUT{}
	// bifSUT := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// 	var start, reset, stop starlark.Value
	// 	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
	// 		"start", &start,
	// 		"reset", &reset,
	// 		"stop", &stop,
	// 	); err != nil {
	// 		return nil, err
	// 	}
	// 	xs, ok := starListOfStringsToSlice(start)
	// 	if !ok {
	// 		return nil, fmt.Errorf("%s(%s = ...) must be a %s", "SUT", "start", "list of strings")
	// 	}
	// 	valSUT.Start = xs
	// 	if xs, ok = starListOfStringsToSlice(reset); !ok {
	// 		return nil, fmt.Errorf("%s(%s = ...) must be a %s", "SUT", "reset", "list of strings")
	// 	}
	// 	valSUT.Reset = xs
	// 	if xs, ok = starListOfStringsToSlice(stop); !ok {
	// 		return nil, fmt.Errorf("%s(%s = ...) must be a %s", "SUT", "stop", "list of strings")
	// 	}
	// 	valSUT.Stop = xs
	// 	return starlark.None, nil
	// }

	// valSpec := Spec{}
	// bifSpec := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// 	var (
	// 		version          = 1
	// 		model, overrides starlark.Value
	// 	)
	// 	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
	// 		"model", &model,
	// 		"overrides", &overrides,
	// 		"version?", &version,
	// 	); err != nil {
	// 		return nil, err
	// 	}
	// 	valSpec.Version = version

	// 	var valOverrides map[string]string
	// 	{
	// 		ovs, ok := overrides.(*starlark.Dict)
	// 		if !ok {
	// 			return nil, fmt.Errorf("%s(%s = ...) must be a %s", "Spec", "overrides", "map of strings to strings")
	// 		}
	// 		valOverrides = make(map[string]string, ovs.Len())
	// 		for _, kv := range ovs.Items() {
	// 			k, ok := kv.Index(0).(starlark.String)
	// 			if !ok {
	// 				return nil, fmt.Errorf("%s(%s = ...) must be a %s", "Spec", "overrides", "map of strings to strings")
	// 			}
	// 			v, ok := kv.Index(1).(starlark.String)
	// 			if !ok {
	// 				return nil, fmt.Errorf("%s(%s = ...) must be a %s", "Spec", "overrides", "map of strings to strings")
	// 			}
	// 			valOverrides[k.GoString()] = v.GoString()
	// 		}
	// 	}
	// 	valSpec.Overrides = valOverrides

	// 	var valModel Modeler
	// 	{
	// 		mo, ok := model.(*starlark.Dict)
	// 		if !ok {
	// 			return nil, fmt.Errorf("%s(%s = ...) must be %s", "Spec", "model", "OpenAPIv3(...)")
	// 		}
	// 		v, found, err := mo.Get(starlark.String("ModelKind"))
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if !found {
	// 			return nil, fmt.Errorf("%s(%s = ...) is incorrect", "Spec", "model")
	// 		}
	// 		vv, ok := v.(starlark.Int)
	// 		if !ok {
	// 			return nil, fmt.Errorf("%s(%s = ...) is incorrect", "Spec", "model")
	// 		}
	// 		vvv, ok := vv.Int64()
	// 		if !ok {
	// 			return nil, fmt.Errorf("%s(%s = ...) is incorrect", "Spec", "model")
	// 		}
	// 		m, ok := registeredIRModels[ModelKind(vvv)]
	// 		if !ok {
	// 			return nil, fmt.Errorf("unexpected model id: %d", vvv)
	// 		}
	// 		if valModel, err = m.GoModelerConstructor(*mo); err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// 	valSpec.Model = valModel

	// 	return starlark.None, nil
	// }

	bifStateF := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		//FIXME: impl State funcs + initial `State` read & map[string]Value enforcement
		// ColorERR.Printf(">>> kwargs = %+v\n", kwargs)
		if err := userRTLang.ModelState.SetKey(starlark.String("blip"), starlark.String("Ah!")); err != nil {
			return nil, err
		}
		return userRTLang.ModelState, nil
	}
	for _, fname := range []string{"StateGet", "StateUpdate"} {
		userRTLang.Globals[fname] = starlark.NewBuiltin(fname, bifStateF)
	}
	userRTLang.Globals[tEnv] = starlark.NewBuiltin(tEnv, bEnv)
	userRTLang.Globals[tTriggerActionAfterProbe] = starlark.NewBuiltin(tTriggerActionAfterProbe, bTriggerActionAfterProbe)
	userRTLang.Thread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	userRTLang.EnvRead = make(map[string]string)
	userRTLang.Triggers = make([]triggerActionAfterProbe, 0)
	userRTLang.InitialRun = true
	defer func() { userRTLang.InitialRun = false }()
	if userRTLang.Globals, err = starlark.ExecFile(userRTLang.Thread, localCfg, nil, userRTLang.Globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	delete(userRTLang.Globals, tEnv)
	delete(userRTLang.Globals, tTriggerActionAfterProbe)
	// TODO: ensure only lowercase things are exported
	userRTLang.ModelState = starlark.NewDict(0)
	if state, ok := userRTLang.Globals[tState]; ok {
		d, ok := state.(*starlark.Dict)
		if !ok {
			panic("FIXME")
		}
		// TODO: check state is a dict with string keys
		for _, kd := range d.Items() {
			k, v := kd.Index(0), kd.Index(1)
			if k.Type() != "string" {
				panic("TODO")
			}
			if err := userRTLang.ModelState.SetKey(k, v); err != nil {
				panic(err)
			}
		}
	}

	// FIXME: ensure Spec + SUT were called
	log.Println("[NFO] starlark cfg globals:", len(userRTLang.Globals.Keys()))
	ColorERR.Printf(">>> globals: %#v\n", userRTLang.Globals)
	return
}

func starListOfStringsToSlice(val starlark.Value) ([]string, bool) {
	if list, ok := val.(*starlark.List); ok {
		n := list.Len()
		xs := make([]string, 0, n)
		for i := 0; i < n; i++ {
			if x, ok := list.Index(i).(starlark.String); ok {
				xs = append(xs, x.GoString())
			} else {
				return nil, false
			}
		}
		return xs, true
	}
	return nil, false
}

func parseCfg(config []byte, showCfg bool) (cfg *UserCfg, err error) {
	var vsn struct {
		V interface{} `yaml:"version"`
	}
	if vsnErr := yaml.Unmarshal(config, &vsn); vsnErr != nil {
		const errFmt = "field 'version' missing! Try `version: %d`"
		err = fmt.Errorf(errFmt, lastCfgVersion)
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}

	version, ok := vsn.V.(int)
	if !ok || !knownVersion(version) {
		err = fmt.Errorf("bad version: `%#v'", vsn.V)
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}

	type cfgParser func(config []byte, showCfg bool) (cfg *UserCfg, err error)
	cfgParsers := []cfgParser{
		parseCfgV001,
	}

	return cfgParsers[version-1](config, showCfg)
}

func knownVersion(v int) bool {
	return 0 < v && v <= lastCfgVersion
}

func parseCfgV001(config []byte, showCfg bool) (cfg *UserCfg, err error) {
	var userConf struct {
		Version uint32   `yaml:"version"`
		Start   []string `yaml:"start"`
		Reset   []string `yaml:"reset"`
		Stop    []string `yaml:"stop"`
		Spec    struct {
			File           string       `yaml:"file"`
			Kind           string       `yaml:"kind"`
			KindIdentified UserCfg_Kind `yaml:"-"`
			Host           string       `yaml:"host"`
			HeaderAuthz    *string      `yaml:"authorization"`
		} `yaml:"spec"`
	}

	if err = yaml.UnmarshalStrict(config, &userConf); err != nil {
		log.Println("[ERR]", err)
		ColorERR.Println("Failed to parse", LocalCfg)
		r := strings.NewReplacer("not found", "unknown")
		for _, e := range strings.Split(err.Error(), "\n") {
			if end := strings.Index(e, " in type struct"); end != -1 {
				ColorERR.Println(r.Replace(e[:end]))
			}
		}
		return
	}

	expectedKind := UserCfg_OpenAPIv3
	if userConf.Spec.Kind != expectedKind.String() {
		err = errors.New("spec's kind must be set to OpenAPIv3")
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}
	userConf.Spec.KindIdentified = expectedKind

	if userConf.Spec.Host == "" {
		def := defaultCfgHost
		log.Printf("[NFO] field 'host' is empty/unset: using %q\n", def)
		userConf.Spec.Host = def
	}
	if !strings.Contains(userConf.Spec.Host, "{{") {
		if _, err = url.ParseRequestURI(userConf.Spec.Host); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}

	if userConf.Spec.HeaderAuthz != nil {
		addHeaderAuthorization = userConf.Spec.HeaderAuthz
	}

	if showCfg {
		ColorNFO.Println("Config:")
		enc := yaml.NewEncoder(os.Stderr)
		defer enc.Close()
		if err = enc.Encode(userConf); err != nil {
			log.Println("[ERR]", err)
			ColorERR.Printf("Failed to pretty-print %s: %#v\n", LocalCfg, err)
			return
		}
	}

	cfg = &UserCfg{
		Version: userConf.Version,
		File:    userConf.Spec.File,
		Kind:    userConf.Spec.KindIdentified,
		Runtime: &UserCfg_Runtime{
			Host: userConf.Spec.Host,
		},
		Exec: &UserCfg_Exec{
			Start:  userConf.Start,
			Reset_: userConf.Reset,
			Stop:   userConf.Stop,
		},
	}
	return
}

func (cfg *UserCfg) script(kind ExecKind) []string {
	return map[ExecKind][]string{
		ExecKind_start: cfg.Exec.Start,
		ExecKind_reset: cfg.Exec.Reset_,
		ExecKind_stop:  cfg.Exec.Stop,
	}[kind]
}

// FindThenReadBlob looks for configured spec and reads it
func (cfg *UserCfg) FindThenReadBlob() (docPath string, blob []byte, err error) {
	//TODO: force relative paths & nested under workdir. Watch out for links
	docPath = cfg.File
	if docPath == `` {
		err = errors.New("Path to spec is empty")
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}

	log.Println("[NFO] reading spec from", docPath)
	if blob, err = ioutil.ReadFile(docPath); err != nil {
		log.Println("[ERR]", err)
		ColorERR.Printf("Could not read '%s'\n", docPath)
	}
	return
}
