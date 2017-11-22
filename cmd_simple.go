package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"gopkg.in/aymerick/raymond.v2"
)

const (
	timeoutShort = 200 * time.Millisecond
	timeoutLong  = 10 * time.Minute
)

type simpleCmd struct {
	V   uint   `json:"v"`
	Cmd string `json:"cmd"`
}

type simpleCmdRep struct {
	Cmd   string  `json:"cmd"`
	V     uint    `json:"v"`
	Us    uint64  `json:"us"`
	HAR   har     `json:"har"`
	Error *string `json:"error"`
}

func (cmd simpleCmd) Kind() string {
	return cmd.Cmd
}

func (cmd simpleCmd) Exec(cfg *ymlCfg) []byte {
	cmdRet := executeScript(cfg, cmd.Kind())
	if isHARReady() {
		cmdRet.HAR = readHAR()
	}
	rep, err := json.Marshal(cmdRet)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}
	clearHAR()

	return rep
}

func executeScript(cfg *ymlCfg, kind string) *simpleCmdRep {
	cmds := cfg.Script[kind]
	if len(cmds) == 0 {
		return &simpleCmdRep{V: 1, Cmd: kind}
	}

	// Note: exec.Cmd fails to cancel with non-*os.File outputs on linux
	//   https://github.com/golang/go/issues/18874
	ctx, cancel := context.WithTimeout(context.Background(), timeoutLong)
	defer cancel()

	var script, stderr bytes.Buffer
	envSerializedPath := pwdID + ".env"
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
	log.Printf("[DBG] $ %s\n", script.Bytes())

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

func snapEnv(envSerializedPath string) (err error) {
	envFile, err := os.OpenFile(envSerializedPath, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer envFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeoutShort)
	defer cancel()

	var script bytes.Buffer
	fmt.Fprintln(&script, "declare -p")
	exe := exec.CommandContext(ctx, shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = envFile
	log.Printf("[DBG] $ %s\n", script.Bytes())

	if err := exe.Run(); err != nil {
		log.Println("[ERR]", err)
		return err
	}
	return nil
}

func readEnv(envVar string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutShort)
	defer cancel()

	cmd := "source " + pwdID + ".env >/dev/null 2>&1 " +
		"&& set -o nounset " +
		"&& echo -n $" + envVar
	var stdout bytes.Buffer
	exe := exec.CommandContext(ctx, shell(), "-c", cmd)
	exe.Stdout = &stdout
	log.Printf("[DBG] $ %s\n", cmd)

	if err := exe.Run(); err != nil {
		log.Println("[ERR]", err)
		return ""
	}
	return string(stdout.Bytes())
}

func shell() string {
	return "/bin/bash"
}

func unstacheEnv(envVar string, options *raymond.Options) raymond.SafeString {
	envVal := readEnv(envVar)
	if envVal == "" {
		fmt.Printf("Environment variable $%s is unset or empty\n", envVar)
		log.Fatal("[ERR] unset or empty env ", envVar)
	}
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
	if cfg.FinalHost == "" || kind != "reset" {
		cfg.FinalHost = unstache(cfg.Host)
	}

	if cfg.FinalPort == "" || kind != "reset" {
		cfg.FinalPort = unstache(cfg.Port)
	}
}
