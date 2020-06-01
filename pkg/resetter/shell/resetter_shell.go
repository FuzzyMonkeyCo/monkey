package resetter_shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
)

const (
	timeoutShort = 200 * time.Millisecond
	timeoutLong  = 2 * time.Minute
)

var _ resetter.Interface = (*Shell)(nil)

// Shell implements resetter.Interface
type Shell struct {
	fm.Clt_Fuzz_Resetter_Shell

	isNotFirstRun bool

	setReadonlyEnvs func(Y io.Writer)
}

// ToProto TODO
func (s *Shell) ToProto() *fm.Clt_Fuzz_Resetter {
	return &fm.Clt_Fuzz_Resetter{
		Resetter: &fm.Clt_Fuzz_Resetter_Shell_{
			Shell: &s.Clt_Fuzz_Resetter_Shell,
		}}
}

// Env TODO
func (s *Shell) Env(read map[string]string) {
	s.setReadonlyEnvs = func(Y io.Writer) {
		for k, v := range read {
			fmt.Fprintf(Y, "declare -p %s >/dev/null 2>&1 || declare -r %s=%s\n", k, k, v)
		}
	}
}

// ExecStart TODO
func (s *Shell) ExecStart(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool) error {
	return s.exec(ctx, stdout, stderr, s.Start)
}

// ExecReset TODO
func (s *Shell) ExecReset(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool) error {
	if only {
		// Makes $ monkey exec reset run as if in between tests
		s.isNotFirstRun = true
	}

	cmds, err := s.commands()
	if err != nil {
		return err
	}

	if !s.isNotFirstRun {
		s.isNotFirstRun = true
	}

	return s.exec(ctx, stdout, stderr, cmds)
}

// ExecStop TODO
func (s *Shell) ExecStop(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool) error {
	return s.exec(ctx, stdout, stderr, s.Stop)
}

// Terminate cleans up after resetter
func (s *Shell) Terminate(ctx context.Context, only bool) error {
	// TODO: maybe run s.Stop
	if err := os.Remove(cwid.EnvFile()); err != nil {
		if !os.IsNotExist(err) {
			log.Println("[ERR]", err)
			return err
		}
	}
	return nil
}

func (s *Shell) commands() (cmds string, err error) {
	switch {
	case len(s.Start) == 0 && len(s.Rst) != 0 && len(s.Stop) == 0:
		log.Println("[NFO] running Shell.Rst")
		cmds = s.Rst
		return

	case len(s.Start) != 0 && len(s.Rst) != 0 && len(s.Stop) != 0:
		if s.isNotFirstRun {
			log.Println("[NFO] running Shell.Rst")
			cmds = s.Rst
			return
		}

		log.Println("[NFO] running Shell.Start then Shell.Rst")
		cmds = s.Start + "\n" + s.Rst
		return

	case len(s.Start) != 0 && len(s.Rst) == 0 && len(s.Stop) != 0:
		if s.isNotFirstRun {
			log.Println("[NFO] running Shell.Stop then Shell.Start")
			cmds = s.Stop + "\n" + s.Start
			return
		}

		log.Println("[NFO] running Shell.Start")
		cmds = s.Start
		return

	default:
		err = errors.New("unhandled Reset() case")
		log.Println("[ERR]", err)
		return
	}
}

func (s *Shell) exec(ctx context.Context, stdout io.Writer, stderr io.Writer, cmds string) (err error) {
	if len(cmds) == 0 {
		err = errors.New("no usable script")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutLong)
	defer cancel()

	envFile := cwid.EnvFile()
	var fi os.FileInfo
	if fi, err = os.Stat(envFile); err != nil {
		if !os.IsNotExist(err) {
			log.Println("[ERR]", err)
			return
		}

		if err = s.snapEnv(ctx, envFile); err != nil {
			return
		}
		if fi, err = os.Stat(envFile); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
	originalMTime := fi.ModTime()

	scriptFile := cwid.ScriptFile()
	var scriptListing bytes.Buffer
	{
		var script *os.File
		if script, err = os.OpenFile(scriptFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0740); err != nil {
			log.Println("[ERR]", err)
			return
		}
		defer script.Close()

		Y := io.MultiWriter(script, &scriptListing)
		s.setReadonlyEnvs(Y)
		fmt.Fprintln(Y, "source", envFile, ">/dev/null 2>&1")
		fmt.Fprintln(Y, "set -o errexit")
		fmt.Fprintln(Y, "set -o errtrace")
		fmt.Fprintln(Y, "set -o nounset")
		fmt.Fprintln(Y, "set -o pipefail")
		fmt.Fprintln(Y, "set -o xtrace")
		fmt.Fprintln(Y, cmds)
		fmt.Fprintln(Y, "set +o xtrace")
		fmt.Fprintln(Y, "set +o pipefail")
		fmt.Fprintln(Y, "set +o nounset")
		fmt.Fprintln(Y, "set +o errtrace")
		fmt.Fprintln(Y, "set +o errexit")
		fmt.Fprintln(Y, "declare -p >", envFile)
	}
	defer os.Remove(scriptFile)

	// NOTE: if piping script to Bash and the script calls exec,
	// even in a subshell, bash will stop execution.
	var stdboth bytes.Buffer
	exe := exec.CommandContext(ctx, s.shell(), "--norc", "--", scriptFile)
	exe.Stdin = nil
	exe.Stdout = io.MultiWriter(&stdboth, stdout)
	exe.Stderr = io.MultiWriter(&stdboth, stderr)
	log.Printf("[DBG] executing script within %s:\n%s", timeoutLong, scriptListing.Bytes())

	ch := make(chan error)
	start := time.Now()
	// https://github.com/golang/go/issues/18874
	//   exec.Cmd fails to cancel with non-*os.File outputs on linux
	// Workaround: race for ctx.Done()
	if err = exe.Start(); err != nil {
		log.Println("[ERR]", err)
		return
	}
	go func() {
		e := exe.Wait()
		ch <- e
		log.Println("[ERR] shell script execution error:", e)
		cancel()
	}()
	select {
	case err = <-ch:
	case <-ctx.Done():
		if err = ctx.Err(); err == context.Canceled {
			return
		}
	}
	log.Printf("[NFO] exec'd in %s", time.Since(start))
	if err != nil {
		reason := stdboth.String() + "\n" + err.Error()
		err = resetter.NewError(strings.Split(reason, "\n"))
		return
	}

	for i, line := range bytes.Split(stdboth.Bytes(), []byte{'\n'}) {
		log.Printf("[NFO] STDERR+STDOUT:%d: %q", i, line)
	}

	if fi, err = os.Stat(envFile); err != nil {
		log.Println("[ERR]", err)
		return
	}
	if fi.ModTime() == originalMTime {
		err = errors.New("make sure to run code that uses `exec` in a (subshell)")
		err = fmt.Errorf("script did not run to completion: %v", err)
		log.Println("[ERR]", err)
		return
	}

	// Whole script ran without error
	return
}

func (s *Shell) snapEnv(ctx context.Context, envSerializedPath string) (err error) {
	envFile, err := os.OpenFile(envSerializedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer envFile.Close()

	ctx, cancel := context.WithTimeout(ctx, timeoutShort)
	defer cancel()

	var script bytes.Buffer
	fmt.Fprintln(&script, "declare -p") // bash specific
	exe := exec.CommandContext(ctx, s.shell(), "--", "/dev/stdin")
	exe.Stdin = &script
	exe.Stdout = envFile
	log.Printf("[DBG] executing script within %s:\n%s", timeoutShort, script.Bytes())

	if err = exe.Run(); err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] snapped env at", envSerializedPath)
	return
}

func (s *Shell) shell() string {
	return "/bin/bash"
}
