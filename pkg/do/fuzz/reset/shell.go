package reset

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

var (
	errSUTShellUndecided = errors.New("unhandled Reset() case in SUTShell")

	_ SUTResetter = (*SUTShell)(nil)
)

// SUTShell TODO
type SUTShell struct {
	fm.Clt_Msg_Fuzz_Resetter_SUTShell
	is_not_first_run bool
}

// ToProto TODO
func (s *SUTShell) ToProto() *fm.Clt_Msg_Fuzz_Resetter {
	return &fm.Clt_Msg_Fuzz_Resetter{
		Resetter: &fm.Clt_Msg_Fuzz_Resetter_SutShell{
			&s.Clt_Msg_Fuzz_Resetter_SUTShell,
		}}
}

// Reset TODO
func (s *SUTShell) Reset(ctx context.Context) error {
	shellCmds, err := s.commands()
	if err != nil {
		return err
	}

	return s.exec(shellCmds)
	// var stderr bytes.Buffer
	// if err = s.executeCommand(&stderr, shellCmds); err != nil {
	// 	fmtExecError(shellCmds, err.Error(), stderr.String())
	// 	return
	// }

	return nil
}

// Terminate TODO
func (s *SUTShell) Terminate(ctx context.Context) error {
	if len(s.Stop) != 0 {
		return s.exec(s.Stop)
	}
	return nil
}

func (s *SUTShell) commands() (cmds string, err error) {
	switch {
	case len(s.Start) == 0 && len(s.Rst) != 0 && len(s.Stop) == 0:
		log.Println("[NFO] running SUTShell.Rst")
		cmds = s.Rst
		return

	case len(s.Start) != 0 && len(s.Rst) != 0 && len(s.Stop) != 0:
		if s.is_not_first_run {
			log.Println("[NFO] running SUTShell.Rst")
			cmds = s.Rst
			return
		}

		log.Println("[NFO] running SUTShell.Start then SUTShell.Rst")
		cmds = s.Start + "\n" + s.Rst
		s.is_not_first_run = true
		return

	case len(s.Start) != 0 && len(s.Rst) == 0 && len(s.Stop) != 0:
		log.Println("[NFO] running SUTShell.Stop then SUTShell.Start")
		cmds = s.Stop + "\n" + s.Start
		return

	default:
		err = errSUTShellUndecided
		log.Println("[ERR]", err)
		return
	}
}

// ExecuteScript TODO
func ExecuteScript(cfg *UserCfg, kind ExecKind) (nxt *RepResetProgress, err error) {
	log.Println("[DBG] >>> exec:", kind.String())
	nxt = &RepResetProgress{Kind: kind}
	shellCmds := cfg.script(kind)
	if len(shellCmds) == 0 {
		return
	}

	wasStopped = kind == ExecKind_stop

	var stderr bytes.Buffer
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
	if err = maybeFinalizeConf(cfg, kind); err != nil {
		HadExecError = true
		return nil, err
	}
	return
}

func executeCommand(nxt *RepResetProgress, stderr *bytes.Buffer, shellCmd string) (err error) {
	const (
		timeoutShort = 200 * time.Millisecond
		timeoutLong  = 2 * time.Minute
	)

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
		e := fmt.Errorf("Timed out after %s", time.Since(start))
		ch <- e
		log.Println("[DBG]", e)
	}()
	go func() {
		e := exe.Run()
		ch <- e
		log.Println("[DBG] execution error:", e)
	}()
	err = <-ch
	nxt.TsDiff += uint64(time.Since(start))

	if err != nil {
		log.Println("[ERR]", stderr.String()+"\n"+err.Error())
		return
	}
	log.Println("[NFO]", stderr.String())
	return
}

func fmtExecError(k ExecKind, i int, c, e, s string) {
	kind := k.String()
	fmt.Printf("Command #%d failed during step '%s' with %s\n", i, kind, e)
	fmt.Printf("Command:\n%s\n", c)
	fmt.Printf("Stderr:\n%s\n", s)
	fmt.Printf("Note that your commands are run with %s", Shell())
	fmt.Println(" along with some shell flags.")
	fmt.Printf("If you're curious, have a look at %s\n", LogID())
	fmt.Printf("And the dumped environment %s\n", EnvID())
}

// SnapEnv TODO
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

// Shell TODO
func Shell() string {
	return "/bin/bash"
}
