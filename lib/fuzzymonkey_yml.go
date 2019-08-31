package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"go.starlark.net/starlark"
	// "go.starlark.net/starlarktest"
	"gopkg.in/yaml.v2"
)

const (
	// LocalCfg is the path of Monkey's config file
	LocalCfg = ".fuzzymonkey.yml"

	lastCfgVersion = 1
	defaultCfgHost = "http://localhost:3000"
)

var (
	// FIXME: find a way to carry some non-proto state from config to Monkey
	addHeaderAuthorization *string
)

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

var registeredIRModels = map[ModelKind]struct {
	StarlarkModelerConstructor string
	GoModelerConstructor       func(starlark.Dict) (Modeler, error)
}{
	ModelKindOpenAPIv3: {
		StarlarkModelerConstructor: `
def OpenAPIv3(**kwargs):
	model = {
		'ModelKind': ` + strconv.Itoa(int(ModelKindOpenAPIv3)) + `,
		'file': kwargs.pop('file'),
	}
	if len(kwargs) != 0:
		fail("Unexpected arguments to Spec():", kwargs)
	return model
`,
		GoModelerConstructor: func(d starlark.Dict) (Modeler, error) {
			mo := ModelOpenAPIv3{}
			{
				key := "file"
				vFile, found, err := d.Get(starlark.String(key))
				if err != nil {
					return nil, err
				}
				if !found {
					return nil, fmt.Errorf("key %q missing", key)
				}
				file, ok := vFile.(starlark.String)
				if !ok {
					return nil, fmt.Errorf("key %q must be a string, was: %v", key, vFile.Type())
				}
				mo.File = file.GoString()
			}
			return mo, nil
		},
	},
}

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

var (
	starlarkThread     *starlark.Thread
	starlarkGlobals    starlark.StringDict
	starlarkModelState *starlark.Dict
)

func loadCfg(config []byte, showCfg bool) (globals starlark.StringDict, err error) {
	const (
		localCfg    = "fuzzymonkey.star"
		preludeStar = "fm-prelude.star"
	)
	var preludeData string

	for _, model := range registeredIRModels {
		preludeData += model.StarlarkModelerConstructor
		preludeData += "\n\n"
	}
	// if globals, err = starlarktest.LoadAssertModule(); err != nil {
	// 	log.Println("[ERR]", err)
	// 	return
	// }

	if globals, err = starlark.ExecFile(&starlark.Thread{
		Name:  "prelude",
		Print: func(_ *starlark.Thread, msg string) { log.Println(msg) },
	}, preludeStar, preludeData, globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	if keys := globals.Keys(); len(keys) != len(registeredIRModels) {
		gs := make([]string, 0, len(keys))
		for _, name := range keys {
			v := globals[name]
			gs = append(gs, fmt.Sprintf("%s (%s) = %s", name, v.Type(), v.String()))
		}
		err = fmt.Errorf("unmatched globals: %+v", strings.Join(gs, "; "))
		log.Println("[ERR]", err)
		return
	}

	valSUT := SUT{}
	bifSUT := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var start, reset, stop starlark.Value
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"start", &start,
			"reset", &reset,
			"stop", &stop,
		); err != nil {
			return nil, err
		}
		xs, ok := starListOfStringsToSlice(start)
		if !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a %s", "SUT", "start", "list of strings")
		}
		valSUT.Start = xs
		if xs, ok = starListOfStringsToSlice(reset); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a %s", "SUT", "reset", "list of strings")
		}
		valSUT.Reset = xs
		if xs, ok = starListOfStringsToSlice(stop); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a %s", "SUT", "stop", "list of strings")
		}
		valSUT.Stop = xs
		return starlark.None, nil
	}

	valSpec := Spec{}
	bifSpec := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			version          = 1
			model, overrides starlark.Value
		)
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"model", &model,
			"overrides", &overrides,
			"version?", &version,
		); err != nil {
			return nil, err
		}
		valSpec.Version = version

		var valOverrides map[string]string
		{
			ovs, ok := overrides.(*starlark.Dict)
			if !ok {
				return nil, fmt.Errorf("%s(%s = ...) must be a %s", "Spec", "overrides", "map of strings to strings")
			}
			valOverrides = make(map[string]string, ovs.Len())
			for _, kv := range ovs.Items() {
				k, ok := kv.Index(0).(starlark.String)
				if !ok {
					return nil, fmt.Errorf("%s(%s = ...) must be a %s", "Spec", "overrides", "map of strings to strings")
				}
				v, ok := kv.Index(1).(starlark.String)
				if !ok {
					return nil, fmt.Errorf("%s(%s = ...) must be a %s", "Spec", "overrides", "map of strings to strings")
				}
				valOverrides[k.GoString()] = v.GoString()
			}
		}
		valSpec.Overrides = valOverrides

		var valModel Modeler
		{
			mo, ok := model.(*starlark.Dict)
			if !ok {
				return nil, fmt.Errorf("%s(%s = ...) must be %s", "Spec", "model", "OpenAPIv3(...)")
			}
			v, found, err := mo.Get(starlark.String("ModelKind"))
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("%s(%s = ...) is incorrect", "Spec", "model")
			}
			vv, ok := v.(starlark.Int)
			if !ok {
				return nil, fmt.Errorf("%s(%s = ...) is incorrect", "Spec", "model")
			}
			vvv, ok := vv.Int64()
			if !ok {
				return nil, fmt.Errorf("%s(%s = ...) is incorrect", "Spec", "model")
			}
			m, ok := registeredIRModels[ModelKind(vvv)]
			if !ok {
				return nil, fmt.Errorf("unexpected model id: %d", vvv)
			}
			if valModel, err = m.GoModelerConstructor(*mo); err != nil {
				return nil, err
			}
		}
		valSpec.Model = valModel

		return starlark.None, nil
	}

	valAfter := make([]struct{ Probe, Predicate, Match, Action starlark.Value }, 0)
	bifAfter := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var probe, predicate, match, action starlark.Value
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"probe", &probe,
			"predicate", &predicate,
			"match", &match,
			"action", &action,
		); err != nil {
			return nil, err
		}

		ColorERR.Printf("probe:%+v, predicate:%+v, match:%+v, action:%+v\n",
			probe, predicate, match, action)
		ColorERR.Printf(">>> After: %#v\n", valAfter)
		return starlark.None, nil
	}

	bifStateF := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		//FIXME: impl State funcs + initial `State` read & map[string]Value enforcement
		ColorERR.Printf(">>> kwargs = %+v\n", kwargs)
		if err := starlarkModelState.SetKey(starlark.String("blip"), starlark.String("Ah!")); err != nil {
			return nil, err
		}
		return starlark.None, nil
	}
	for _, fname := range []string{"StateGet", "StateUpdate"} {
		globals[fname] = starlark.NewBuiltin(fname, bifStateF)
	}

	globals["SUT"] = starlark.NewBuiltin("SUT", bifSUT)
	globals["Spec"] = starlark.NewBuiltin("Spec", bifSpec)
	globals["After"] = starlark.NewBuiltin("After", bifAfter)
	muh := "BigBoyState"
	boi := starlark.NewDict(42)
	if err := boi.SetKey(starlark.String("blip"), starlark.String("Ah!")); err != nil {
		panic(err)
	}
	globals[muh] = boi
	ColorERR.Printf(">>> %s: %#v\n", muh, globals[muh].String())
	starlarkThread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { ColorNFO.Println(msg) },
	}
	if starlarkGlobals, err = starlark.ExecFile(starlarkThread, localCfg, nil, globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	ColorERR.Printf(">>> Spec: %#v\n", valSpec)
	ColorERR.Printf(">>> SUT: %#v\n", valSUT)
	// FIXME: ensure Spec + SUT were called & cleanup globals maybe
	log.Println("[NFO] starlark cfg globals:", len(globals.Keys()))
	ColorERR.Printf(">>> globals: %#v\n", globals)

	// ColorERR.Printf(">>> %s: %#v\n", muh, globals[muh].String())
	// f, ok := globals["unpure"].(*starlark.Function)
	// if !ok {
	// 	panic(err)
	// }
	// ret, err := starlark.Call(th1, f, starlark.Tuple{}, []starlark.Tuple{})
	// ColorERR.Printf(">>> called ret: %#v\n", ret)
	// ColorERR.Printf(">>> called err: %#v\n", err)
	// ColorERR.Printf(">>> %s: %#v\n", muh, globals[muh].String())
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
