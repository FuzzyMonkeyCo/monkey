package pkg

import (
	"io/ioutil"
	"log"

	"github.com/pkg/errors"
	// yaml "gopkg.in/yaml.v2"
)

const (
	// lastCfgVersion = 1
	defaultCfgHost = "http://localhost:3000"
)

// FIXME: use new Modeler intf struc to pass these
var addHeaderAuthorization, addHost *string

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
