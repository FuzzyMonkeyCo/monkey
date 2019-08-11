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
func (m *ModelOpenAPIv3) Kind() ModelKind { return ModelKindOpenAPIv3 }
func (m *ModelOpenAPIv3) pp()             { fmt.Println(m) }

// SUT describes ways to reset the system under test to a known initial state
type SUT struct {
	Start, Reset, Stop []string
}

func loadCfg(config []byte, showCfg bool) (cfg *UserCfg, err error) {
	const (
		localCfg    = ".fuzzymonkey.star"
		preludeStar = "fm-prelude.star"
		preludeData = `
def SUT(**kwargs):
	sut = {
		'start': kwargs.pop('start', []),
		'reset': kwargs.pop('reset', []),
		'stop':  kwargs.pop('stop', []),
	}
	if len(kwargs) != 0:
		fail("Unexpected arguments to SUT():", kwargs)
	for k, xs in sut.items():
		if not (type(xs) == 'list' and all([type(x) == 'string' for x in xs])):
			fail("SUT({} = ...) must be a list of strings".format(k))
	return sut

def Spec(**kwargs):
	spec = {
		'version': kwargs.pop('version', 1),
		'model': kwargs.pop('model'),
		'overrides': kwargs.pop('overrides', {}),
	}
	if len(kwargs) != 0:
		fail("Unexpected arguments to Spec():", kwargs)
	if type(spec.version) != 'int':
		fail("Spec(version = ...) must be a positive integer")
	if type(spec.model) != 'dict':
		fail("Spec(model = ...) must be a Model object")
	if type(spec.overrides) != 'dict':
		fail("Spec(overrides = ...) must be a dict")
	return spec

def OpenAPIv3(**kwargs):
	model = {}
	if len(kwargs) != 0:
		fail("Unexpected arguments to Spec():", kwargs)

`
	)

	var globals starlark.StringDict
	if globals, err = starlark.ExecFile(&starlark.Thread{
		Name:  "prelude",
		Print: func(_ *starlark.Thread, msg string) { log.Println(msg) },
	}, preludeStar, preludeData, starlark.StringDict{}); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	fmt.Println("\nGlobals:")
	for _, name := range globals.Keys() {
		v := globals[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}

	var valSUT SUT
	bifSUT := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var start, reset, stop *starlark.List
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"start", &start,
			"reset", &reset,
			"stop", &stop,
		); err != nil {
			return nil, err
		}
		ColorWRN.Printf("%+v\n", start)
		ColorWRN.Printf("%+v\n", reset)
		ColorWRN.Printf("%+v\n", stop)
		valSUT = SUT{
			// Start: start,
			// Reset: reset,
			// Stop:  stop,
		}
		ColorWRN.Printf("%+v\n", valSUT)
		var ret *starlark.Dict = starlark.NewDict(3)
		ret.SetKey(starlark.String("start"), start)
		ret.SetKey(starlark.String("reset"), reset)
		ret.SetKey(starlark.String("stop"), stop)
		return starlark.None, nil
	}

	var valSpec Spec
	bifSpec := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			version   = 1
			model     *starlark.Dict
			overrides *starlark.Dict
		)
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"model", &model,
			"overrides", &overrides,
			"version?", &version,
		); err != nil {
			return nil, err
		}
		ColorWRN.Printf("%+v\n", model)
		var valModel Modeler

		valOverrides := make(map[string]string, overrides.Len())
		for _, kv := range overrides.Items() {
			k := kv.Index(0).(starlark.String).GoString()
			v := kv.Index(1).(starlark.String).GoString()
			valOverrides[k] = v
		}

		valSpec = Spec{
			Version:   version,
			Model:     valModel,
			Overrides: valOverrides,
		}
		ColorERR.Printf("%+v\n", valSpec)
		return starlark.None, nil
	}

	bifOpenAPIv3 := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var file string
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"file", &file,
		); err != nil {
			return nil, err
		}
		ret := starlark.NewDict(2)
		ret.SetKey(starlark.String("ModelKind"), starlark.MakeInt(int(ModelKindOpenAPIv3)))
		ret.SetKey(starlark.String("file"), starlark.String(file))
		return ret, nil
	}

	globals["SUT"] = starlark.NewBuiltin("SUT", bifSUT)
	globals["Spec"] = starlark.NewBuiltin("Spec", bifSpec)
	globals["OpenAPIv3"] = starlark.NewBuiltin("OpenAPIv3", bifOpenAPIv3)
	if globals, err = starlark.ExecFile(&starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { ColorNFO.Println(msg) },
	}, localCfg, nil, globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	// Print the global environment.
	fmt.Println("\nGlobals:")
	for _, name := range globals.Keys() {
		v := globals[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}
	return
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
