package lib

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"text/template"
	"time"
)

const (
	timeoutShort = 200 * time.Millisecond
	timeoutLong  = 2 * time.Minute
)

var (
	isRunning = false
	// To exit with statusFailedExec
	HadExecError = false
	// To not post-stop after stop
	wasStopped = false
)

func (act *ReqDoReset) exec(mnk *Monkey) (err error) {
	if isHARReady() {
		/// exec of FuzzProgress
		// var str string
		// if *cmd.Passed {
		// 	str = "✓"
		// } else {
		// 	if !*cmd.Passed {
		// 		str = "✗"
		// 	}
		// }

		// if cmd.ShrinkingFrom != nil {
		// 	shrinkingFrom = *cmd.ShrinkingFrom
		// 	if lastLane.T == cmd.ShrinkingFrom.T {
		// 		str += "\n"
		// 	}
		// }
		// fmt.Print(str)

		clearHAR()
	}

	nxt := ExecuteScript(mnk.Cfg, act.GetKind())
	if err = mnk.ws.cast(nxt); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func ExecuteScript(cfg *UserCfg, kind ExecKind) (nxt *RepResetProgress) {
	log.Println("[DBG] >>> exec:", ExecKind_name[int32(kind)])
	nxt = &RepResetProgress{Kind: kind}
	shellCmds := cfg.script(kind)
	if len(shellCmds) == 0 {
		return
	}

	wasStopped = kind == ExecKind_stop

	var stderr bytes.Buffer
	var err error
	for i, shellCmd := range shellCmds {
		if err = executeCommand(nxt, &stderr, shellCmd); err != nil {
			fmtExecError(kind, i+1, shellCmd, err.Error(), stderr.String())
			HadExecError = true
			nxt.Failure = true
			return
		}
	}
	nxt.Success = true

	isRunning = kind != ExecKind_stop
	maybeFinalizeConf(cfg, kind)
	return
}

func executeCommand(nxt *RepResetProgress, stderr *bytes.Buffer, shellCmd string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutLong)
	defer cancel()

	var script bytes.Buffer
	fmt.Fprintln(&script, "source", EnvID(), ">/dev/null 2>&1")
	fmt.Fprintln(&script, "set -o errexit")
	fmt.Fprintln(&script, "set -o errtrace")
	fmt.Fprintln(&script, "set -o nounset")
	fmt.Fprintln(&script, "set -o pipefail")
	fmt.Fprintln(&script, "set -o xtrace")
	fmt.Fprintln(&script, shellCmd)
	fmt.Fprintln(&script, "set +o xtrace")
	fmt.Fprintln(&script, "set +o pipefail")
	fmt.Fprintln(&script, "set +o nounset")
	fmt.Fprintln(&script, "set +o errtrace")
	fmt.Fprintln(&script, "set +o errexit")
	fmt.Fprintln(&script, "declare -p >", EnvID())

	exe := exec.CommandContext(ctx, Shell(), "--", "/dev/stdin")
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
		log.Println("[DBG] execution error:", error)
	}()
	err = <-ch
	nxt.Usec += uint64(time.Since(start) / time.Microsecond)

	if err != nil {
		log.Println("[ERR]", stderr.String()+"\n"+err.Error())
		return
	}
	log.Println("[NFO]", stderr.String())
	return
}

func fmtExecError(k ExecKind, i int, c, e, s string) {
	kind := ExecKind_name[int32(k)]
	fmt.Printf("Command #%d failed during step '%s' with %s\n", i, kind, e)
	fmt.Printf("Command:\n%s\n", c)
	fmt.Printf("Stderr:\n%s\n", s)
	fmt.Printf("Note that your commands are run with %s", Shell())
	fmt.Println(" along with some shell flags.")
	fmt.Printf("If you're curious, have a look at %s\n", LogID())
	fmt.Printf("And the dumped environment %s\n", EnvID())
}

func SnapEnv(envSerializedPath string) (err error) {
	envFile, err := os.OpenFile(envSerializedPath, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer envFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeoutShort)
	defer cancel()

	var script bytes.Buffer
	fmt.Fprintln(&script, "declare -p") // bash specific
	exe := exec.CommandContext(ctx, Shell(), "--", "/dev/stdin")
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

	cmd := "source " + EnvID() + " >/dev/null 2>&1 " +
		"&& set -o nounset " +
		"&& printf $" + envVar
	var stdout bytes.Buffer
	exe := exec.CommandContext(ctx, Shell(), "-c", cmd)
	exe.Stdout = &stdout
	log.Printf("[DBG] whithin %s $ %s\n", timeoutShort, cmd)

	if err := exe.Run(); err != nil {
		log.Println("[ERR]", err)
		return ""
	}
	return stdout.String()
}

func Shell() string {
	return "/bin/bash"
}

func unstacheEnv(envVar string) (envVal string, err error) {
	envVal = readEnv(envVar)
	if envVal == "" {
		err = fmt.Errorf("Environment variable $%s is unset or empty", envVar)
	}
	return
}

func unstache(field string) string {
	if field[:2] != "{{" {
		return field
	}

	funcMap := template.FuncMap{
		"env": unstacheEnv,
	}
	tmpl := template.New("unstache").Funcs(funcMap)

	var err error
	if tmpl, err = tmpl.Parse(field); err != nil {
		log.Println("[ERR]", err)
		panic(err)
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, ""); err != nil {
		log.Println("[ERR]", err)
		panic(err)
	}
	return buffer.String()
}

func maybeFinalizeConf(cfg *UserCfg, kind ExecKind) {
	if kind == ExecKind_stop {
		return
	}

	var wg sync.WaitGroup
	if cfg.Runtime.FinalHost == "" || kind != ExecKind_reset {
		wg.Add(1)
		go func() {
			cfg.Runtime.FinalHost = unstache(cfg.Runtime.Host)
			wg.Done()
		}()
	}

	if cfg.Runtime.FinalPort == "" || kind != ExecKind_reset {
		wg.Add(1)
		go func() {
			cfg.Runtime.FinalPort = unstache(cfg.Runtime.Port)
			wg.Done()
		}()
	}
	wg.Wait()
}
