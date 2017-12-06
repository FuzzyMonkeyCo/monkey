package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/aymerick/raymond.v2"
)

const (
	timeoutShort = 200 * time.Millisecond
	timeoutLong  = 2 * time.Minute
)

type simpleCmd struct {
	V             uint   `json:"v"`
	Cmd           string `json:"cmd"`
	Passed        *bool  `json:"passed"`
	ShrinkingFrom *lane  `json:"shrinking_from"`
}

type simpleCmdRep struct {
	Cmd    string  `json:"cmd"`
	V      uint    `json:"v"`
	Us     uint64  `json:"us"`
	Reason *string `json:"error"`
}

func (cmd *simpleCmd) Kind() string {
	return cmd.Cmd
}

func (cmd *simpleCmd) Exec(cfg *ymlCfg) (rep []byte, err error) {
	if isHARReady() {
		progress(cmd)
		clearHAR()
	}

	cmdRep, err := executeScript(cfg, cmd.Kind())
	if err != nil {
		return
	}

	rep, err = json.Marshal(cmdRep)
	if err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func progress(cmd *simpleCmd) {
	var str string
	if *cmd.Passed {
		str = "."
	} else {
		if !*cmd.Passed {
			str = "x"
		}
	}

	if cmd.ShrinkingFrom != nil {
		shrinkingFrom = *cmd.ShrinkingFrom
		if lastLane.T == cmd.ShrinkingFrom.T {
			str += "\n"
		}
	}
	fmt.Printf(str)
}

func executeScript(cfg *ymlCfg, kind string) (cmdRep *simpleCmdRep, err error) {
	cmdRep = &simpleCmdRep{V: 1, Cmd: kind}
	shellCmds := cfg.Script[kind]
	if len(shellCmds) == 0 {
		return
	}

	var stderr bytes.Buffer
	for _, shellCmd := range shellCmds {
		if err = executeCommand(cmdRep, &stderr, shellCmd); err != nil {
			fmt.Printf("A command failed during '%s':\n", kind)
			fmtIndented("Command:", shellCmd)
			fmtIndented("Reason:", err.Error())
			return
		}
	}

	maybeFinalizeConf(cfg, kind)
	return
}

func fmtIndented(headline, toIndent string) {
	fmt.Println(headline)
	for _, txt := range strings.Split(toIndent, "\n") {
		fmt.Printf("\t%s\n", txt)
	}
}

func executeCommand(cmdRep *simpleCmdRep, stderr *bytes.Buffer, shellCmd string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutLong)
	defer cancel()

	var script bytes.Buffer
	envSerializedPath := pwdID + ".env"
	fmt.Fprintln(&script, "source", envSerializedPath, ">/dev/null 2>&1")
	fmt.Fprintln(&script, "set -x")
	fmt.Fprintln(&script, "set -o errexit")
	fmt.Fprintln(&script, "set -o errtrace")
	// fmt.Fprintln(&script, "set -o execfail")
	fmt.Fprintln(&script, "set -o nounset")
	fmt.Fprintln(&script, "set -o pipefail")
	fmt.Fprintln(&script, shellCmd)
	fmt.Fprintln(&script, "declare -p >", envSerializedPath)

	exe := exec.CommandContext(ctx, shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = os.Stdout
	exe.Stderr = stderr
	log.Printf("[DBG] within %s $ %s\n", timeoutLong, script.Bytes())

	ch := make(chan error)
	start := time.Now()
	// https://github.com/golang/go/issues/18874
	//   exec.Cmd fails to cancel with non-*os.File outputs on linux
	// Racing for ctx.Done() is a workaround to ^
	go func() {
		<-ctx.Done()
		error := fmt.Errorf("Timed out after %s", time.Since(start))
		ch <- error
		log.Println("[DBG]", error)
	}()
	go func() {
		error := exe.Run()
		ch <- error
		log.Println("[DBG]", error)
	}()
	err = <-ch
	cmdRep.Us += uint64(time.Since(start) / time.Microsecond)

	if err != nil {
		reason := string(stderr.Bytes()) + "\n" + err.Error()
		log.Println("[ERR]", reason)
		cmdRep.Reason = &reason
		return
	}
	log.Println("[NFO]", string(stderr.Bytes()))
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
	log.Printf("[DBG] within %s $ %s\n", timeoutShort, script.Bytes())

	if err = exe.Run(); err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] snapped env at ", envSerializedPath)
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
	log.Printf("[DBG] whithin %s $ %s\n", timeoutShort, cmd)

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
		log.Fatal("[ERR] ", err)
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

	str, err := raymond.Render(field, nil)
	if err != nil {
		log.Panic("[ERR] ", err)
	}
	return str
}

func maybeFinalizeConf(cfg *ymlCfg, kind string) {
	var wg sync.WaitGroup

	if cfg.FinalHost == "" || kind != "reset" {
		wg.Add(1)
		go func() {
			cfg.FinalHost = unstache(cfg.Host)
			wg.Done()
		}()
	}

	if cfg.FinalPort == "" || kind != "reset" {
		wg.Add(1)
		go func() {
			cfg.FinalPort = unstache(cfg.Port)
			wg.Done()
		}()
	}

	wg.Wait()
}
