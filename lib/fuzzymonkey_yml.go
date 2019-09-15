package lib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
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

// Modeler describes checkable models
type Modeler interface {
	SetSUTResetter(SUTResetter)
	GetSUTResetter() SUTResetter

	Pretty(w io.Writer) (n int, err error)
}

// SUTResetter describes ways to reset the system under test to a known initial state
type SUTResetter interface {
	Start(context.Context) error
	Reset(context.Context) error
	Stop(context.Context) error
}

var _ SUTResetter = (*SUTShell)(nil)

// SUTShell TODO
type SUTShell struct {
	start, reset, stop string
}

// Start TODO
func (s *SUTShell) Start(ctx context.Context) error { return nil }

// Reset TODO
func (s *SUTShell) Reset(ctx context.Context) error { return nil }

// Stop TODO
func (s *SUTShell) Stop(ctx context.Context) error { return nil }

// ModelerFunc TODO
type ModelerFunc func(d starlark.StringDict) (Modeler, error)
type slBuiltin func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

var registeredIRModels = map[string]ModelerFunc{
	"OpenAPIv3": func(d starlark.StringDict) (Modeler, error) {
		mo := &ModelOpenAPIv3{}
		var (
			found              bool
			file, host, hAuthz starlark.Value
		)

		if file, found = d["file"]; !found || file.Type() != "string" {
			// TODO: introduce specific error type so as to build `<key>(field = ...)` messages
			return nil, errors.New("OpenAPIv3(file = ...) must be a string")
		}
		mo.File = file.(starlark.String).GoString()

		if host, found = d["host"]; found && host.Type() != "string" {
			return nil, errors.New("OpenAPIv3(host = ...) must be a string")
		}
		if found {
			h := host.(starlark.String).GoString()
			mo.Host = h
			addHost = &h
		}

		if hAuthz, found = d["header_authorization"]; found && hAuthz.Type() != "string" {
			return nil, errors.New("OpenAPIv3(header_authorization = ...) must be a string")
		}
		if found {
			authz := hAuthz.(starlark.String).GoString()
			mo.HeaderAuthorization = authz
			addHeaderAuthorization = &authz
		}

		return mo, nil
	},
}

func modelMaker(modeler ModelerFunc) slBuiltin {
	return func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		fname := b.Name()
		if args.Len() != 0 {
			return nil, fmt.Errorf("%s(...) does not take positional arguments", fname)
		}

		u := make(starlark.StringDict, len(kwargs))
		r := make(starlark.StringDict, len(kwargs))
		for _, kv := range kwargs {
			k, v := kv.Index(0), kv.Index(1)
			key := k.(starlark.String).GoString()
			// FIXME: prempt Exec* fields + other Capitalized keys
			reserved := false
			for i, c := range key {
				if !(c <= unicode.MaxASCII && unicode.IsPrint(c)) {
					panic("FIXME: illegal")
				}
				if i == 0 && unicode.IsUpper(c) {
					reserved = true
					break
				}
			}
			if !reserved {
				u[key] = v
			} else {
				r[key] = v
			}
		}
		mo, err := modeler(u)
		if err != nil {
			return nil, err
		}
		resetter, err := newSUTResetter(fname, r)
		if err != nil {
			return nil, err
		}
		mo.SetSUTResetter(resetter)

		userRTLang.Modelers = append(userRTLang.Modelers, mo)
		return starlark.None, nil
	}
}

func newSUTResetter(modelerName string, r starlark.StringDict) (SUTResetter, error) {
	var (
		ok bool
		v  starlark.Value
		vv starlark.String
		t  string
		// TODO: other SUTResetter.s
		resetter = &SUTShell{}
	)
	t = tExecStart
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.start = vv.GoString()
	}
	t = tExecReset
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.reset = vv.GoString()
	}
	t = tExecStop
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.stop = vv.GoString()
	}
	if len(r) != 0 {
		return nil, fmt.Errorf("Unexpected arguments to %s(): %s", modelerName, strings.Join(r.Keys(), ", "))
	}
	return resetter, nil
}

var _ Modeler = (*ModelOpenAPIv3)(nil)

// ModelOpenAPIv3 describes OpenAPIv3 models
type ModelOpenAPIv3 struct {
	resetter SUTResetter

	/// Fields editable on initial run
	// File is a path within current directory pointing to a YAML spec
	File string
	// Host superseeds the spec's base URL
	Host string
	// HeaderAuthorization if non-empty is added to requests as bearer token
	HeaderAuthorization string

	// FIXME? tcap *tCapHTTP
}

// SetSUTResetter TODO
func (m *ModelOpenAPIv3) SetSUTResetter(sr SUTResetter) { m.resetter = sr }

// GetSUTResetter TODO
func (m *ModelOpenAPIv3) GetSUTResetter() SUTResetter { return m.resetter }

// Pretty TODO
func (m *ModelOpenAPIv3) Pretty(w io.Writer) (int, error) { return fmt.Fprintf(w, "%+v\n", m) }

type modelState struct {
	d *starlark.Dict
}

var (
	_ starlark.Value           = (*modelState)(nil)
	_ starlark.HasAttrs        = (*modelState)(nil)
	_ starlark.HasSetKey       = (*modelState)(nil)
	_ starlark.IterableMapping = (*modelState)(nil)
	_ starlark.Sequence        = (*modelState)(nil)
	_ starlark.Comparable      = (*modelState)(nil)
)

func newModelState(size int) *modelState {
	return &modelState{d: starlark.NewDict(size)}
}
func (s *modelState) Clear() error { return s.d.Clear() }
func (s *modelState) Delete(k starlark.Value) (starlark.Value, bool, error) {
	if !slValuePrintableASCII(k) {
		panic("FIXME: illegal")
	}
	return s.d.Delete(k)
}
func (s *modelState) Get(k starlark.Value) (starlark.Value, bool, error) {
	if !slValuePrintableASCII(k) {
		panic("FIXME: illegal")
	}
	return s.d.Get(k)
}
func (s *modelState) Items() []starlark.Tuple    { return s.d.Items() }
func (s *modelState) Keys() []starlark.Value     { return s.d.Keys() }
func (s *modelState) Len() int                   { return s.d.Len() }
func (s *modelState) Iterate() starlark.Iterator { return s.d.Iterate() }
func (s *modelState) SetKey(k, v starlark.Value) error {
	if !slValuePrintableASCII(k) {
		panic("FIXME: illegal")
	}
	return s.d.SetKey(k, v)
}
func (s *modelState) String() string                           { return s.d.String() }
func (s *modelState) Type() string                             { return "ModelState" }
func (s *modelState) Freeze()                                  { s.d.Freeze() }
func (s *modelState) Truth() starlark.Bool                     { return s.d.Truth() }
func (s *modelState) Hash() (uint32, error)                    { return s.d.Hash() }
func (s *modelState) Attr(name string) (starlark.Value, error) { return s.d.Attr(name) }
func (s *modelState) AttrNames() []string                      { return s.d.AttrNames() }
func (s *modelState) CompareSameType(op syntax.Token, ss starlark.Value, depth int) (bool, error) {
	return s.d.CompareSameType(op, ss, depth)
}
func slValuePrintableASCII(k starlark.Value) bool {
	key, ok := k.(starlark.String)
	if !ok {
		return false
	}
	return printableASCII(key.GoString())
}
func printableASCII(s string) bool {
	l := 0
	for _, c := range s {
		if !(c <= unicode.MaxASCII && unicode.IsPrint(c)) {
			return false
		}
		l++
	}
	if l > 255 {
		return false
	}
	return true
}

var userRTLang struct {
	Thread     *starlark.Thread
	Globals    starlark.StringDict
	ModelState *modelState
	// EnvRead holds all the envs looked up on initial run
	EnvRead  map[string]string
	Triggers []triggerActionAfterProbe

	Modelers []Modeler
}

// TODO: turn these into methods of userRTLang
const (
	tEnv                     = "Env"
	tExecReset               = "ExecReset"
	tExecStart               = "ExecStart"
	tExecStop                = "ExecStop"
	tState                   = "State"
	tTriggerActionAfterProbe = "TriggerActionAfterProbe"
)

func bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()
	// FIXME: actually maybe read env from Exec shell? These shells should inherit user env anyway?
	read, ok := os.LookupEnv(envStr)
	if !ok {
		return def, nil
	}
	userRTLang.EnvRead[envStr] = read
	return starlark.String(read), nil
}

type triggerActionAfterProbe struct {
	Name              starlark.String
	Probe             starlark.Tuple
	Predicate, Action *starlark.Function
}

func bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var trigger triggerActionAfterProbe
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"name?", &trigger.Name,
		"probe", &trigger.Probe,
		"predicate", &trigger.Predicate,
		"action", &trigger.Action,
	); err != nil {
		return nil, err
	}
	// TODO: enforce arities
	log.Println("[NFO] registering", b.Name(), trigger)
	if name := trigger.Name.GoString(); name == "" {
		trigger.Name = starlark.String(trigger.Action.Name())
		// TODO: complain if trigger.Name == "lambda"
	}
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
	userRTLang.Globals[tEnv] = starlark.NewBuiltin(tEnv, bEnv)
	userRTLang.Globals[tTriggerActionAfterProbe] = starlark.NewBuiltin(tTriggerActionAfterProbe, bTriggerActionAfterProbe)
	userRTLang.Thread = &starlark.Thread{
		Name:  "cfg",
		Print: func(_ *starlark.Thread, msg string) { ColorWRN.Println(msg) },
	}
	userRTLang.EnvRead = make(map[string]string)
	userRTLang.Triggers = make([]triggerActionAfterProbe, 0)
	if userRTLang.Globals, err = starlark.ExecFile(userRTLang.Thread, localCfg, nil, userRTLang.Globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	// Ensure at least one model was defined
	ColorERR.Printf(">>> modelers: %v\n", userRTLang.Modelers)
	if len(userRTLang.Modelers) == 0 {
		panic("FIXME")
	}

	ColorERR.Printf(">>> envs: %+v\n", userRTLang.EnvRead)
	ColorERR.Printf(">>> trigs: %+v\n", userRTLang.Triggers)
	delete(userRTLang.Globals, tEnv)
	delete(userRTLang.Globals, tTriggerActionAfterProbe)

	if state, ok := userRTLang.Globals[tState]; ok {
		d, ok := state.(*starlark.Dict)
		if !ok {
			panic("FIXME")
		}
		delete(userRTLang.Globals, tState)
		userRTLang.ModelState = newModelState(d.Len())
		for _, kd := range d.Items() {
			k, v := kd.Index(0), kd.Index(1)
			// Ensure State keys are all String.s
			if !slValuePrintableASCII(k) {
				panic("FIXME")
			}
			// Ensure State values are all literals
			switch v.(type) {
			case starlark.NoneType, starlark.Bool:
			case starlark.Int, starlark.Float:
			case starlark.String:
			case *starlark.List, *starlark.Dict, *starlark.Set:
			default:
				panic("FIXME")
			}
			ColorERR.Printf(">>> modelState: SetKey(%v, %v)\n", k, v)
			// if err := userRTLang.ModelState.SetKey(k, v); err != nil {
			if err := userRTLang.ModelState.SetKey(k, starlark.NewDict(0)); err != nil {
				panic(err)
			}
		}
	} else {
		userRTLang.ModelState = newModelState(0)
	}
	// TODO: ensure only lowercase things are exported
	for key := range userRTLang.Globals {
		if len(key) > 255 {
			panic("FIXME")
		}
		for _, c := range key {
			if unicode.IsUpper(c) {
				panic("FIXME")
			}
			break
		}
		for _, c := range key {
			if !unicode.IsPrint(c) {
				panic("FIXME")
			}
		}
	}
	log.Println("[NFO] starlark cfg globals:", len(userRTLang.Globals.Keys()))
	ColorERR.Printf(">>> globals: %#v\n", userRTLang.Globals)
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
