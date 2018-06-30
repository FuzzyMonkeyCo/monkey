package main

//FIXME: switch to TOML?
// https://github.com/toml-lang/toml
// https://github.com/crdoconnor/strictyaml#why-strictyaml
// https://github.com/pelletier/go-toml

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	localYML       = ".fuzzymonkey.yml"
	lastYMLVersion = 1
	defaultYMLHost = "localhost"
	defaultYMLPort = "3000"
)

func newCfg(yml []byte, showCfg bool) (cfg *YmlCfg, err error) {
	var vsn struct {
		V interface{} `yaml:"version"`
	}
	if vsnErr := yaml.Unmarshal(yml, &vsn); vsnErr != nil {
		err = fmt.Errorf("field 'version' missing! Try `version: %d`",
			lastYMLVersion)
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	version, ok := vsn.V.(int)
	if !ok || !knownVersion(version) {
		err = fmt.Errorf("bad version: `%#v'", vsn.V)
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	type cfgParser func(yml []byte, showCfg bool) (cfg *YmlCfg, err error)
	cfgParsers := []cfgParser{
		newCfgV001,
	}

	return cfgParsers[version-1](yml, showCfg)
}

func knownVersion(v int) bool {
	if 0 < v && v <= lastYMLVersion {
		return true
	}
	return false
}

func newCfgV001(yml []byte, showCfg bool) (cfg *YmlCfg, err error) {
	var ymlConf struct {
		Version uint32   `yaml:"version"`
		Start   []string `yaml:"start"`
		Reset   []string `yaml:"reset"`
		Stop    []string `yaml:"stop"`
		Spec    struct {
			File           string      `yaml:"file"`
			Kind           string      `yaml:"kind"`
			KindIdentified YmlCfg_Kind `yaml:"-"`
			Host           string      `yaml:"host"`
			Port           string      `yaml:"port"`
		} `yaml:"spec"`
	}

	if err = yaml.UnmarshalStrict(yml, &ymlConf); err != nil {
		log.Println("[ERR]", err)
		colorERR.Println("Failed to parse", localYML)
		r := strings.NewReplacer("not found", "unknown")
		for _, e := range strings.Split(err.Error(), "\n") {
			if end := strings.Index(e, " in type struct"); end != -1 {
				colorERR.Println(r.Replace(e[:end]))
			}
		}
		return
	}

	expectedKind := YmlCfg_OpenAPIv3
	if ymlConf.Spec.Kind != YmlCfg_Kind_name[int32(expectedKind)] {
		err = errors.New("spec's kind must be set to OpenAPIv3")
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}
	ymlConf.Spec.KindIdentified = expectedKind

	if ymlConf.Spec.Host == "" {
		def := defaultYMLHost
		log.Printf("[NFO] field 'host' is empty/unset: using %v\n", def)
		ymlConf.Spec.Host = def
	}

	if ymlConf.Spec.Port == "" {
		def := defaultYMLPort
		log.Printf("[NFO] field 'port' is empty/unset: using %v\n", def)
		ymlConf.Spec.Port = def
	}

	if showCfg {
		colorNFO.Println("Config:")
		enc := yaml.NewEncoder(os.Stderr)
		defer enc.Close()
		if err = enc.Encode(ymlConf); err != nil {
			log.Println("[ERR]", err)
			colorERR.Printf("Failed to pretty-print %s: %#v\n", localYML, err)
			return
		}
	}

	cfg = &YmlCfg{
		Version: ymlConf.Version,
		File:    ymlConf.Spec.File,
		Kind:    ymlConf.Spec.KindIdentified,
		Runtime: &YmlCfg_Runtime{
			Host: ymlConf.Spec.Host,
			Port: ymlConf.Spec.Port,
		},
		Exec: &YmlCfg_Exec{
			Start:  ymlConf.Start,
			Reset_: ymlConf.Reset,
			Stop:   ymlConf.Stop,
		},
	}
	return
}

func (cfg *YmlCfg) script(kind cmdKind) []string {
	return map[cmdKind][]string{
		kindStart: cfg.Exec.Start,
		kindReset: cfg.Exec.Reset_,
		kindStop:  cfg.Exec.Stop,
	}[kind]
}

func (cfg *YmlCfg) findThenReadBlob() (docPath string, blob []byte, err error) {
	//TODO: force relative paths & nested under workdir. Watch out for links
	docPath = cfg.File
	if docPath == `` {
		err = errors.New("Path to spec is empty")
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	log.Println("[NFO] reading spec from", docPath)
	if blob, err = ioutil.ReadFile(docPath); err != nil {
		log.Println("[ERR]", err)
		fmt.Printf("Could not read '%s'\n", docPath)
	}
	return
}

func readYML() (yml []byte, err error) {
	fd, err := os.Open(localYML)
	if err != nil {
		log.Println("[ERR]", err)
		colorERR.Printf("You must provide a readable %s file in the current directory.\n", localYML)
		return
	}
	defer fd.Close()

	if yml, err = ioutil.ReadAll(fd); err != nil {
		log.Println("[ERR]", err)
	}
	return
}
