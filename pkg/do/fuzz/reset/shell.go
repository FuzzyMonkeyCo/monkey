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
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
)

const (
	timeoutSUTShellShort = 200 * time.Millisecond
	timeoutSUTShellLong  = 2 * time.Minute
)

var (
	errSUTShellUndecided = errors.New("unhandled Reset() case in SUTShell")
	errSUTShellNoScript          = errors.New("no usable script for SUTShell")

	_ SUTResetter = (*SUTShell)(nil)
)

// SUTShell TODO
type SUTShell struct {
	fm.Clt_Msg_Fuzz_Resetter_SUTShell
	isNotFirstRun bool
}

// ToProto TODO
func (s *SUTShell) ToProto() *fm.Clt_Msg_Fuzz_Resetter {
	return &fm.Clt_Msg_Fuzz_Resetter{
		Resetter: &fm.Clt_Msg_Fuzz_Resetter_SutShell{
			&s.Clt_Msg_Fuzz_Resetter_SUTShell,
		}}
}

// ExecStart TODO
func (s *SUTShell) ExecStart(ctx context.Context, clt fm.Client) error {
	return s.exec(ctx, s.Start)
}

// ExecReset TODO
func (s *SUTShell) ExecReset(ctx context.Context, clt fm.Client) error {
	if clt == nil {
		return s.exec(ctx, s.Rst)
	}

	cmds, err := s.commands()
	if err != nil {
		return err
	}

	if !s.isNotFirstRun {
		s.isNotFirstRun = true
		if _, err := os.Stat(s.shell()); os.IsNotExist(err) {
			err = fmt.Errorf("shell %s is required", s.shell())
			log.Println("[ERR]", err)
			return err
		}
		if err := s.snapEnv(ctx, cwid.EnvFile()); err != nil {
			return err
		}
	}

	return s.exec(ctx, cmds)
}

// ExecStop TODO
func (s *SUTShell) ExecStop(ctx context.Context, clt fm.Client) error {
	return s.exec(ctx, s.Stop)
}

// Terminate cleans up after resetter
func (s *SUTShell) Terminate(ctx context.Context, clt fm.Client) error {
	// TODO: maybe run s.Stop
	return os.Remove(cwid.EnvFile())
}

func (s *SUTShell) commands() (cmds string, err error) {
	switch {
	case len(s.Start) == 0 && len(s.Rst) != 0 && len(s.Stop) == 0:
		log.Println("[NFO] running SUTShell.Rst")
		cmds = s.Rst
		return

	case len(s.Start) != 0 && len(s.Rst) != 0 && len(s.Stop) != 0:
		if s.isNotFirstRun {
			log.Println("[NFO] running SUTShell.Rst")
			cmds = s.Rst
			return
		}

		log.Println("[NFO] running SUTShell.Start then SUTShell.Rst")
		cmds = s.Start + "\n" + s.Rst
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

func (s *SUTShell) exec(ctx context.Context, cmds string) error {
	if len(cmds) == 0 {
		return errSUTShellNoScript
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutSUTShellLong)
	defer cancel()

	var script bytes.Buffer
	fmt.Fprintln(&script, "source", cwid.EnvFile(), ">/dev/null 2>&1")
	fmt.Fprintln(&script, "set -o errexit")
	fmt.Fprintln(&script, "set -o errtrace")
	fmt.Fprintln(&script, "set -o nounset")
	fmt.Fprintln(&script, "set -o pipefail")
	fmt.Fprintln(&script, "set -o xtrace")
	fmt.Fprintln(&script, cmds)
	fmt.Fprintln(&script, "set +o xtrace")
	fmt.Fprintln(&script, "set +o pipefail")
	fmt.Fprintln(&script, "set +o nounset")
	fmt.Fprintln(&script, "set +o errtrace")
	fmt.Fprintln(&script, "set +o errexit")
	fmt.Fprintln(&script, "declare -p >", cwid.EnvFile())

	var stderr bytes.Buffer
	exe := exec.CommandContext(ctx, s.shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = os.Stdout // TODO: plug Progresser here
	exe.Stderr = &stderr // TODO: same as above
	log.Printf("[DBG] within %s $ %s\n", timeoutSUTShellLong, script.Bytes())

	ch := make(chan error)
	// https://github.com/golang/go/issues/18874
	//   exec.Cmd fails to cancel with non-*os.File outputs on linux
	// Racing for ctx.Done() is a workaround to ^
	go func() {
		<-ctx.Done()
		e := errors.New("script timed out")
		ch <- e
		log.Println("[DBG]", e)
	}()
	go func() {
		e := exe.Run()
		ch <- e
		log.Println("[DBG] execution error:", e)
	}()
	if err := <-ch; err != nil {
		// TODO: mux stderr+stdout and fwd to server to track progress
		reason := stderr.String()+"\n"+err.Error()
		log.Println("[ERR]", reason)
		return NewError(strings.Split(reason, "\n"))
	}
	log.Println("[NFO]", stderr.String())
	return nil
}

func (s *SUTShell) snapEnv(ctx context.Context, envSerializedPath string) (err error) {
	envFile, err := os.OpenFile(envSerializedPath, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer envFile.Close()

	ctx, cancel := context.WithTimeout(ctx, timeoutSUTShellShort)
	defer cancel()

	var script bytes.Buffer
	fmt.Fprintln(&script, "declare -p") // bash specific
	exe := exec.CommandContext(ctx, s.shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = envFile
	log.Printf("[DBG] within %s $ %s\n", timeoutSUTShellShort, script.Bytes())

	if err = exe.Run(); err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] snapped env at ", envSerializedPath)
	return
}

func (s *SUTShell)shell() string {
	return "/bin/bash"
}
