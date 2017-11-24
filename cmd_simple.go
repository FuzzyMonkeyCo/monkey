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

func (cmd simpleCmd) Exec(cfg *ymlCfg) (rep []byte, err error) {
	cmdRep, err := executeScript(cfg, cmd.Kind())
	if err != nil {
		return
	}

	if isHARReady() {
		fmt.Printf(".")
		cmdRep.HAR = readHAR()
	}
	rep, err = json.Marshal(cmdRep)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	clearHAR()

	return
}

func executeScript(cfg *ymlCfg, kind string) (cmdRep *simpleCmdRep, err error) {
	cmdRep = &simpleCmdRep{V: 1, Cmd: kind}
	shellCmds := cfg.Script[kind]
	if len(shellCmds) == 0 {
		return
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
	for _, shellCmd := range shellCmds {
		fmt.Fprintln(&script, shellCmd)
	}
	fmt.Fprintln(&script, "declare -p >", envSerializedPath)

	exe := exec.CommandContext(ctx, shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = os.Stdout
	exe.Stderr = &stderr
	log.Printf("[DBG] $ %s\n", script.Bytes())

	start := time.Now()
	err = exe.Run()
	cmdRep.Us = uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		error := string(stderr.Bytes()) + "\n" + err.Error()
		log.Println(error)
		cmdRep.Error = &error
		return
	}

	err = maybeF1inalizeConf(cfg, kind)
	return
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

	if err = exe.Run(); err != nil {
		log.Println("[ERR]", err)
	}
	return
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
		err := fmt.Errorf("Environment variable $%s is unset or empty", envVar)
		fmt.Println(err)
		log.Panic("[ERR] ", err)
	}
	return raymond.SafeString(envVal)
}

func unstacheInit() {
	raymond.RegisterHelper("env", unstacheEnv)
}

func unstache(field string) (str string, err error) {
	if field[:2] != "{{" {
		return field, nil
	}

	str, err = raymond.Render(field, nil)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if "" == str {
		err = fmt.Errorf("Mustache field '%s' was resolved to the empty string", field)
		log.Println("[ERR]", err)
		fmt.Println(err)
	}
	return
}

func maybeF1inalizeConf(cfg *ymlCfg, kind string) (err error) {
	var host, port string

	if cfg.FinalHost == "" || kind != "reset" {
		host, err = unstache(cfg.Host)
		if err != nil {
			return
		}
		cfg.FinalHost = host
	}

	if cfg.FinalPort == "" || kind != "reset" {
		port, err = unstache(cfg.Port)
		if err != nil {
			return
		}
		cfg.FinalPort = port
	}

	return
}
