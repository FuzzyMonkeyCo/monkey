package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"os/exec"
	"time"

	"gopkg.in/aymerick/raymond.v2"
)

type simpleCmd struct {
	V   uint   `json:"v"`
	Cmd string `json:"cmd"`
}

type simpleCmdRep struct {
	Cmd   string  `json:"cmd"`
	V     uint    `json:"v"`
	Us    uint64  `json:"us"`
	Error *string `json:"error"`
}

func (cmd simpleCmd) Kind() string {
	return cmd.Cmd
}

func (cmd simpleCmd) Exec(cfg *ymlCfg) []byte {
	cmdRet := executeScript(cfg, cmd.Kind())
	rep, err := json.Marshal(cmdRet)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}
	return rep
}

func executeScript(cfg *ymlCfg, kind string) *simpleCmdRep {
	cmds := cfg.Script[kind]
	if len(cmds) == 0 {
		return &simpleCmdRep{V: 1, Cmd: kind}
	}

	cmdTimeout := 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	var script, stderr bytes.Buffer
	envSerializedPath := uniquePath()
	fmt.Fprintln(&script, "source", envSerializedPath, ">/dev/null 2>&1")
	fmt.Fprintln(&script, "set -x")
	fmt.Fprintln(&script, "set -o errexit")
	fmt.Fprintln(&script, "set -o errtrace")
	fmt.Fprintln(&script, "set -o nounset")
	fmt.Fprintln(&script, "set -o pipefail")
	for _, cmd := range cmds {
		fmt.Fprintln(&script, cmd)
	}
	fmt.Fprintln(&script, "declare -p >", envSerializedPath)

	exe := exec.CommandContext(ctx, shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = os.Stdout
	exe.Stderr = &stderr

	log.Printf("$ %s\n", script.Bytes())
	start := time.Now()
	err := exe.Run()
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		error := string(stderr.Bytes()) + "\n" + err.Error()
		log.Println(error)
		return &simpleCmdRep{V: 1, Cmd: kind, Us: us, Error: &error}
	}

	maybeF1inalizeConf(cfg, kind)

	return &simpleCmdRep{V: 1, Cmd: kind, Us: us}
}

func uniquePath() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	h := fnv.New64a()
	h.Write([]byte(cwd))
	return "/tmp/" + coveredci + "_" + fmt.Sprintf("%d", h.Sum64()) + ".env"
}

func snapEnv(envSerializedPath string) {
	cmdTimeout := 200 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := "declare -p >" + envSerializedPath
	exe := exec.CommandContext(ctx, shell(), "-c", cmd)
	log.Printf("$ %s\n", cmd)

	if err := exe.Run(); err != nil {
		log.Fatal("[ERR] ", err)
	}
}

//FIXME: make this faster! parse the .env file?
func readEnv(envSerializedPath, envVar string) string {
	cmdTimeout := 200 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := "source " + envSerializedPath + " >/dev/null 2>&1 && echo -n " + envVar
	var stdout bytes.Buffer
	exe := exec.CommandContext(ctx, shell(), "-c", cmd)
	exe.Stdout = &stdout
	log.Printf("$ %s\n", cmd)

	if err := exe.Run(); err != nil {
		log.Fatal("[ERR] ", err)
	}
	return string(stdout.Bytes())
}

func shell() string {
	return "/bin/bash"
}

func unstacheEnv(envVar string, options *raymond.Options) raymond.SafeString {
	envVal := readEnv(uniquePath(), "$"+envVar)
	return raymond.SafeString(envVal)
}

func unstacheInit() {
	raymond.RegisterHelper("env", unstacheEnv)
}

func unstache(field string) string {
	if field[:2] != "{{" {
		return field
	}

	result, err := raymond.Render(field, nil)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}
	if "" == result {
		log.Fatalf("[ERR] Mustache field '%s' was resolved to the empty string\n", field)
	}
	return result
}

func maybeF1inalizeConf(cfg *ymlCfg, kind string) {
	if kind == "start" || kind == "stop" {
		cfg.FinalHost = unstache(cfg.Host)
		cfg.FinalPort = unstache(cfg.Port)
	}
}
