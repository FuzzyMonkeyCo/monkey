package shell

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
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"go.starlark.net/starlark"
)

// Name names the Starlark builtin
const Name = "shell"

const (
	timeoutShort = 200 * time.Millisecond
	timeoutLong  = 2 * time.Minute
)

// New instanciates a new resetter
func New(kwargs []starlark.Tuple) (resetter.Interface, error) {
	var name, start, reset, stop starlark.String
	var provides tags.UniqueStringsNonEmpty
	if err := starlark.UnpackArgs(Name, nil, kwargs,
		"name", &name,
		"provides", &provides,
		// TODO: waiton = "tcp/4000", various recipes => 1 rsttr / service
		"start??", &start,
		"reset??", &reset,
		"stop??", &stop,
	); err != nil {
		return nil, err
	}
	s := &Resetter{
		name:     name.GoString(),
		provides: provides.GoStrings(),
	}
	s.Start = start.GoString()
	s.Rst = reset.GoString()
	s.Stop = stop.GoString()
	return s, nil
}

var _ resetter.Interface = (*Resetter)(nil)

// Resetter implements resetter.Interface
type Resetter struct {
	name     string
	provides []string
	fm.Clt_Fuzz_Resetter_Shell

	isNotFirstRun bool

	setReadonlyEnvs func(Y io.Writer)
}

// Name uniquely identifies this instance
func (s *Resetter) Name() string { return s.name }

// Provides lists the models a resetter resets
func (s *Resetter) Provides() []string { return s.provides }

// ToProto marshals a resetter.Interface implementation into a *fm.Clt_Fuzz_Resetter
func (s *Resetter) ToProto() *fm.Clt_Fuzz_Resetter {
	return &fm.Clt_Fuzz_Resetter{
		Name:     s.name,
		Provides: s.provides,
		Resetter: &fm.Clt_Fuzz_Resetter_Shell_{
			Shell: &s.Clt_Fuzz_Resetter_Shell,
		}}
}

// Env passes envs read during startup
func (s *Resetter) Env(read map[string]string) {
	s.setReadonlyEnvs = func(Y io.Writer) {
		for k, v := range read {
			fmt.Fprintf(Y, "declare -p %s >/dev/null 2>&1 || declare -r %s=%s\n", k, k, v)
		}
	}
}

// ExecStart executes the setup phase of the System Under Test
func (s *Resetter) ExecStart(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool) error {
	return s.exec(ctx, stdout, stderr, s.Start)
}

// ExecReset resets the System Under Test to a state similar to a post-ExecStart state
func (s *Resetter) ExecReset(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool) error {
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

// ExecStop executes the cleanup phase of the System Under Test
func (s *Resetter) ExecStop(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool) error {
	return s.exec(ctx, stdout, stderr, s.Stop)
}

// Terminate cleans up after a resetter.Interface implementation instance
func (s *Resetter) Terminate(ctx context.Context, stdout io.Writer, stderr io.Writer) (err error) {
	if hasStop := strings.TrimSpace(s.Stop) != ""; hasStop {
		if err = s.ExecStop(ctx, stdout, stderr, true); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}

	if err = os.Remove(cwid.EnvFile()); err != nil {
		if !os.IsNotExist(err) {
			log.Println("[ERR]", err)
			return
		}
		err = nil
	}
	return
}

func (s *Resetter) commands() (cmds string, err error) {
	var (
		hasStart = strings.TrimSpace(s.Start) != ""
		hasReset = strings.TrimSpace(s.Rst) != ""
		hasStop  = strings.TrimSpace(s.Stop) != ""
	)
	switch {
	case !hasStart && hasReset && !hasStop:
		log.Println("[NFO] running Shell.Rst")
		cmds = s.Rst
		return

	case hasStart && hasReset && hasStop:
		if s.isNotFirstRun {
			log.Println("[NFO] running Shell.Rst")
			cmds = s.Rst
			return
		}

		log.Println("[NFO] running Shell.Start then Shell.Rst")
		cmds = s.Start + "\n" + s.Rst
		return

	case hasStart && !hasReset && hasStop:
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

func (s *Resetter) exec(ctx context.Context, stdout io.Writer, stderr io.Writer, cmds string) (err error) {
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
		if f := s.setReadonlyEnvs; f != nil {
			f(Y)
		} else {
			log.Println("[NFO] setReadonlyEnvs was nil")
		}
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

func (s *Resetter) snapEnv(ctx context.Context, envSerializedPath string) (err error) {
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

func (s *Resetter) shell() string {
	return "/bin/bash"
}
