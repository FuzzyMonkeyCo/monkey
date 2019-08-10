package lib

//FIXME: switch to TOML?
// https://github.com/toml-lang/toml
// https://github.com/crdoconnor/strictyaml#why-strictyaml
// https://github.com/pelletier/go-toml

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

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

	if _, err = loadCfg(config, showCfg); err != nil {
		return
	}

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
	// repeat(str, n=1) is a Go function called from Starlark.
	// It behaves like the 'string * int' operation.
	repeat := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var s string
		n := 1
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, "s", &s, "n?", &n); err != nil {
			return nil, err
		}
		return starlark.String(strings.Repeat(s, n)), nil
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
		ColorWRN.Printf("%+v\n", overrides)
		valOverrides := make(map[string]string, overrides.Len())
		for _, kv := range overrides.Items() {
			k := kv.Index(0).(starlark.String).GoString()
			v := kv.Index(1).(starlark.String).GoString()
			valOverrides[k] = v
		}
		valSpec = Spec{
			Version: version,
			// Model:     model,
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
		ColorWRN.Printf("%+v\n", SUT{
			// Start: start,
			// Reset: reset,
			// Stop:  stop,
		})
		var ret *starlark.Dict = starlark.NewDict(3)
		ret.SetKey(starlark.String("start"), start)
		ret.SetKey(starlark.String("reset"), reset)
		ret.SetKey(starlark.String("stop"), stop)
		return ret, nil
	}

	// The Thread defines the behavior of the built-in 'print' function.
	thread := &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { ColorNFO.Println(msg) },
	}

	const localCfg = ".fuzzymonkey.star"
	var globals starlark.StringDict
	if globals, err = starlark.ExecFile(thread, localCfg, nil, starlark.StringDict{
		"greeting":  starlark.String("hello"),
		"repeat":    starlark.NewBuiltin("repeat", repeat),
		"Spec":      starlark.NewBuiltin("Spec", bifSpec),
		"OpenAPIv3": starlark.NewBuiltin("OpenAPIv3", bifOpenAPIv3),
		"SUT":       starlark.NewBuiltin("SUT", bifSUT),
	}); err != nil {
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
