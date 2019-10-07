package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	// "net/url"
	"os"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	// yaml "gopkg.in/yaml.v2"
	"github.com/FuzzyMonkeyCo/monkey/pkg"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

const (
	// lastCfgVersion = 1
	defaultCfgHost = "http://localhost:3000"
)

// FIXME: use new Modeler intf struc to pass these
var addHeaderAuthorization, addHost *string

func (mnk *monkey) loadCfg(localCfg string, showCfg bool) (err error) {
	if mnk.globals, err = starlark.ExecFile(mnk.thread, localCfg, nil, mnk.globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	// Ensure at least one model was defined
	ColorERR.Printf(">>> modelers: %v\n", mnk.modelers)
	if len(mnk.modelers) == 0 {
		err = errors.New("no modelers are registered")
		log.Println("[ERR]", err)
		return
	}

	ColorERR.Printf(">>> envs: %+v\n", mnk.envRead)
	ColorERR.Printf(">>> trigs: %+v\n", mnk.triggers)
	delete(mnk.globals, tEnv)
	delete(mnk.globals, tTriggerActionAfterProbe)

	if state, ok := mnk.globals[tState]; ok {
		d, ok := state.(*starlark.Dict)
		if !ok {
			err = fmt.Errorf("monkey State must be a dict, got: %s", state.Type())
			log.Println("[ERR]", err)
			return
		}
		delete(mnk.globals, tState)
		mnk.modelState = newModelState(d.Len())
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
			ColorERR.Printf(">>> modelState: SetKey(%v, %v)\n", k, v)
			var vv starlark.Value
			if vv, err = slValueCopy(v); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if err = mnk.modelState.SetKey(k, vv); err != nil {
				log.Println("[ERR]", err)
				return
			}
		}
	} else {
		mnk.modelState = newModelState(0)
	}
	for key := range mnk.globals {
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
	log.Println("[NFO] starlark cfg globals:", len(mnk.globals.Keys()))
	ColorERR.Printf(">>> globals: %#v\n", mnk.globals)
	return
}

// func parseCfg(config []byte, showCfg bool) (cfg *UserCfg, err error) {
// 	var vsn struct {
// 		V interface{} `yaml:"version"`
// 	}
// 	if vsnErr := yaml.Unmarshal(config, &vsn); vsnErr != nil {
// 		const errFmt = "field 'version' missing! Try `version: %d`"
// 		err = fmt.Errorf(errFmt, lastCfgVersion)
// 		log.Println("[ERR]", err)
// 		ColorERR.Println(err)
// 		return
// 	}

// 	version, ok := vsn.V.(int)
// 	if !ok || !knownVersion(version) {
// 		err = fmt.Errorf("bad version: `%#v'", vsn.V)
// 		log.Println("[ERR]", err)
// 		ColorERR.Println(err)
// 		return
// 	}

// 	type cfgParser func(config []byte, showCfg bool) (cfg *UserCfg, err error)
// 	cfgParsers := []cfgParser{
// 		parseCfgV001,
// 	}

// 	return cfgParsers[version-1](config, showCfg)
// }

// func knownVersion(v int) bool {
// 	return 0 < v && v <= lastCfgVersion
// }

// func parseCfgV001(config []byte, showCfg bool) (cfg *UserCfg, err error) {
// 	var userConf struct {
// 		Version uint32   `yaml:"version"`
// 		Start   []string `yaml:"start"`
// 		Reset   []string `yaml:"reset"`
// 		Stop    []string `yaml:"stop"`
// 		Spec    struct {
// 			File           string       `yaml:"file"`
// 			Kind           string       `yaml:"kind"`
// 			KindIdentified UserCfg_Kind `yaml:"-"`
// 			Host           string       `yaml:"host"`
// 			HeaderAuthz    *string      `yaml:"authorization"`
// 		} `yaml:"spec"`
// 	}

// 	if err = yaml.UnmarshalStrict(config, &userConf); err != nil {
// 		log.Println("[ERR]", err)
// 		ColorERR.Println("Failed to parse", LocalCfg)
// 		r := strings.NewReplacer("not found", "unknown")
// 		for _, e := range strings.Split(err.Error(), "\n") {
// 			if end := strings.Index(e, " in type struct"); end != -1 {
// 				ColorERR.Println(r.Replace(e[:end]))
// 			}
// 		}
// 		return
// 	}

// 	expectedKind := UserCfg_OpenAPIv3
// 	if userConf.Spec.Kind != expectedKind.String() {
// 		err = errors.New("spec's kind must be set to OpenAPIv3")
// 		log.Println("[ERR]", err)
// 		ColorERR.Println(err)
// 		return
// 	}
// 	userConf.Spec.KindIdentified = expectedKind

// 	if userConf.Spec.Host == "" {
// 		def := defaultCfgHost
// 		log.Printf("[NFO] field 'host' is empty/unset: using %q\n", def)
// 		userConf.Spec.Host = def
// 	}
// 	if !strings.Contains(userConf.Spec.Host, "{{") {
// 		if _, err = url.ParseRequestURI(userConf.Spec.Host); err != nil {
// 			log.Println("[ERR]", err)
// 			return
// 		}
// 	}

// 	if userConf.Spec.HeaderAuthz != nil {
// 		addHeaderAuthorization = userConf.Spec.HeaderAuthz
// 	}

// 	if showCfg {
// 		ColorNFO.Println("Config:")
// 		enc := yaml.NewEncoder(os.Stderr)
// 		defer enc.Close()
// 		if err = enc.Encode(userConf); err != nil {
// 			log.Println("[ERR]", err)
// 			ColorERR.Printf("Failed to pretty-print %s: %#v\n", LocalCfg, err)
// 			return
// 		}
// 	}

// 	cfg = &UserCfg{
// 		Version: userConf.Version,
// 		File:    userConf.Spec.File,
// 		Kind:    userConf.Spec.KindIdentified,
// 		Runtime: &UserCfg_Runtime{
// 			Host: userConf.Spec.Host,
// 		},
// 		Exec: &UserCfg_Exec{
// 			Start:  userConf.Start,
// 			Reset_: userConf.Reset,
// 			Stop:   userConf.Stop,
// 		},
// 	}
// 	return
// }

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
