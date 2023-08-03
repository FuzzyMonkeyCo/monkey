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
	"strconv"
	"strings"
	"sync"
	"time"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
)

// Name names the Starlark builtin
const Name = "shell"

// TODO:{start,reset,strop}_file a la Bazel
// write files to /tmp once + chmodx

const (
	shell = "/bin/bash" // TODO: use mentioned shell

	scriptTimeout = 2 * time.Minute // TODO: tune through kwargs
)

// New instanciates a new resetter
func New(kwargs []starlark.Tuple) (resetter.Interface, error) {
	var name, start, reset, stop starlark.String
	var provides tags.UniqueStringsNonEmpty
	if err := starlark.UnpackArgs(Name, nil, kwargs,
		"name", &name,
		"provides", &provides,
		// TODO: waiton = "tcp/4000", various recipes => 1 rsttr per service
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

	scriptsCreator sync.Once
	scriptsPaths   map[shellCmd]string
	stdin          io.WriteCloser
	// stdboth        bytes.Buffer
	sherr chan error
	rcoms *rcoms
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

// ExecStart executes the setup phase of the System Under Test
func (s *Resetter) ExecStart(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool, envRead map[string]string) error {
	return s.exec(ctx, stdout, stderr, envRead, cmdStart)
}

// ExecReset resets the System Under Test to a state similar to a post-ExecStart state
func (s *Resetter) ExecReset(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool, envRead map[string]string) error {
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

	return s.exec(ctx, stdout, stderr, envRead, cmds...)
}

// ExecStop executes the cleanup phase of the System Under Test
func (s *Resetter) ExecStop(ctx context.Context, stdout io.Writer, stderr io.Writer, only bool, envRead map[string]string) error {
	return s.exec(ctx, stdout, stderr, envRead, cmdStop)
}

// TidyOutput filter maps over each line
func (s *Resetter) TidyOutput(stdeither [][]byte) resetter.TidiedOutput {
	// for
	return stdeither
}

// Terminate cleans up after a resetter.Interface implementation instance
func (s *Resetter) Terminate(ctx context.Context, stdout io.Writer, stderr io.Writer, envRead map[string]string) (err error) {
	if hasStop := strings.TrimSpace(s.Stop) != ""; hasStop {
		if err = s.ExecStop(ctx, stdout, stderr, true, envRead); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
	log.Println("[NFO] exiting shell singleton")
	s.signal("###:exit:", "")
	// close(s.sherr)
	// s.rcoms = nil
	return
}

type shellCmd int

const (
	cmdStart shellCmd = iota
	cmdReset
	cmdStop
)

func (cmd shellCmd) String() string {
	return map[shellCmd]string{
		cmdStart: "Start",
		cmdReset: "Reset",
		cmdStop:  "Stop",
	}[cmd]
}

func (s *Resetter) commands() (cmds []shellCmd, err error) {
	var (
		hasStart = "" != strings.TrimSpace(s.Start)
		hasReset = "" != strings.TrimSpace(s.Rst)
		hasStop  = "" != strings.TrimSpace(s.Stop)
	)
	switch {
	case !hasStart && hasReset && !hasStop:
		log.Println("[NFO] running Shell.Rst")
		cmds = []shellCmd{cmdReset}
		return

	case hasStart && hasReset && hasStop:
		if s.isNotFirstRun {
			log.Println("[NFO] running Shell.Rst")
			cmds = []shellCmd{cmdReset}
			return
		}

		log.Println("[NFO] running Shell.Start then Shell.Rst")
		cmds = []shellCmd{cmdStart, cmdReset}
		return

	case hasStart && !hasReset && hasStop:
		if s.isNotFirstRun {
			log.Println("[NFO] running Shell.Stop then Shell.Start")
			cmds = []shellCmd{cmdStop, cmdStart}
			return
		}

		log.Println("[NFO] running Shell.Start")
		cmds = []shellCmd{cmdStart}
		return

	default:
		err = errors.New("missing at least `shell( reset = \"...code...\" )`")
		log.Println("[ERR]", err)
		return
	}
}

func (s *Resetter) exec(ctx context.Context, stdout, stderr io.Writer, envRead map[string]string, cmds ...shellCmd) (err error) {
	if len(cmds) == 0 {
		err = errors.New("no usable script")
		return
	}

	s.scriptsCreator.Do(func() {
		paths := make([]string, 0, 3)
		s.scriptsPaths = make(map[shellCmd]string, 3)
		for cmd, command := range map[shellCmd]struct {
			Name, Code string
		}{
			cmdStart: {"start", s.Start},
			cmdReset: {"reset", s.Rst},
			cmdStop:  {"stop", s.Stop},
		} {
			path := fmt.Sprintf("%s%s.bash", cwid.Prefixed(), command.Name)
			if err = writeScript(path, command.Name, command.Code, envRead); err != nil {
				log.Println("[ERR]", err)
				return
			}
			s.scriptsPaths[cmd] = path
			paths = append(paths, path)
		}

		main := fmt.Sprintf("%s%s.bash", cwid.Prefixed(), "main")
		if err = writeMainScript(main, paths); err != nil {
			return
		}

		s.sherr = make(chan error, 1)
		// TODO: isolate shell better. See: https://github.com/maxmcd/bramble/blob/205f61427fe505d109d22ef94967561006d6c83d/internal/command/cli.go#L258
		exe := exec.CommandContext(ctx, shell, "--norc", "--", main)
		stdin, err := exe.StdinPipe()
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
		s.stdin = stdin
		s.rcoms = &rcoms{errcodes: make(chan uint8)}
		go func() {
			var stdboth bytes.Buffer

			// TODO: mux stderr+stdout and fwd to server to track progress
			exe.Stdout = io.MultiWriter(&stdboth, stdout, s.rcoms) //FIXME: drop our prefixed intructions
			exe.Stderr = io.MultiWriter(&stdboth, stderr, s.rcoms) //FIXME: drop our prefixed intructions
			log.Printf("[DBG] starting shell instance")

			// go func() {
			// 	// fixme: turn into progresswriter
			// 	log.Printf("[NFO] STDERR+STDOUT: %q", stdboth.String())
			// 	for i, line := range bytes.Split(stdboth.Bytes(), []byte{'\n'}) {
			// 		log.Printf("[NFO] STDERR+STDOUT:%d: %q", i, line)
			// 	}
			// }()

			// go func() {
			// defer close(s.sherr)
			// defer func() { s.rcoms = nil }()
			defer log.Println("[NFO] shell singleton exited")

			if err := exe.Run(); err != nil {
				log.Println("[ERR] forwarding error:", err)
				////////////////////////////////s.stdboth.Close()
				exe.Stdout = stdout
				exe.Stderr = stderr
				// data := stdboth.Bytes()

				// log.Println("[ERR] building then forwarding resetter error:", err)

				reason := stdboth.String() + "\n" + err.Error()
				var lines [][]byte
				for _, line := range strings.Split(reason, "\n") {
					if strings.HasPrefix(line, stdeitherPrefixSkip) {
						continue
					}
					if x := strings.TrimPrefix(line, stdeitherPrefixDropPrefix); x != line {
						lines = append(lines, []byte(x))
					}
				}
				s.sherr <- resetter.NewError(lines)
				// s.sherr <- err
				// // close(s.sherr)
			}
		}()
	})
	if err != nil {
		return
	}

	for _, cmd := range cmds {
		if err = s.execEach(ctx, cmd); err != nil {
			return
		}
	}
	return
}

const (
	stdeitherPrefixSkip       = "+ "
	stdeitherPrefixDropPrefix = "++ "
)

func writeMainScript(name string, paths []string) (err error) {
	var script *os.File
	if script, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0740); err != nil {
		return
	}
	defer script.Close()

	fmt.Fprintln(script, `#!`+shell+` -ux

set -o pipefail

trap 'echo Shell instance exiting >&2' EXIT
trap 'rm -f "`+strings.Join(append(paths, name), `" "`)+`"' EXIT

while read -r x; do
	case "$x" in
	'###:exec:'*) x=${x:9} ;;
	'###:exit:') exit 0 ;;
	*) echo "###:unexpected:$x" && exit 42 ;;
	esac

	if ! script=$(cat "$x"); then
		# File was deleted
		break
	fi

	if [[ -z "$script" ]]; then
		# No new input yet
		sleep 0.1
		continue
	fi

	source "$x"
	echo "###:exitcode:$?"
done
`)
	return
}

// FIXME: generalize from progress_writer
// see https://stackoverflow.com/a/42208606/1418165

var _ io.Writer = (*wrp)(nil)

type wrp struct {
	w io.Writer
}

func wrap(stdio io.Writer) io.Writer {
	return &wrp{w: stdio}
}

func (w *wrp) Write(p []byte) (int, error) {
	do := func(data []byte) {
		if n := len(data); n > 0 {
			if x := bytes.TrimPrefix(data, []byte("++ ")); n != len(x) {
				if string(x) != "set +o xtrace" {
					w.w.Write(x)
				}
			}
		}
	}

	for i := 0; ; {
		n := bytes.IndexAny(p[i:], "\n\r")
		if n < 0 {
			do(p[i:])
			break
		}
		do(p[i : i+n])
		i += n + 1
	}
	return len(p), nil
}

var _ io.Writer = (*rcoms)(nil)

type rcoms struct {
	errcodes chan uint8
}

func (coms *rcoms) Write(p []byte) (int, error) {
	do := func(data []byte) {
		if n := len(data); n > 0 {
			if x := bytes.TrimPrefix(data, []byte("###:exitcode:")); n != len(x) {
				if y, err := strconv.ParseInt(string(x), 10, 8); err == nil {
					coms.errcodes <- uint8(y)
				}
			}
			if x := bytes.TrimPrefix(data, []byte("###:unexpected:")); n != len(x) {
				log.Printf("[ERR] unexpected rcoms: %s", x)
			}
		}
	}

	for i := 0; ; {
		n := bytes.IndexAny(p[i:], "\n\r")
		if n < 0 {
			do(p[i:])
			break
		}
		do(p[i : i+n])
		i += n + 1
	}
	return len(p), nil
}

func writeScript(scriptFile, cmdName, code string, envRead map[string]string) (err error) {
	var script *os.File
	if script, err = os.OpenFile(scriptFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0740); err != nil {
		return
	}
	defer script.Close()

	fmt.Fprintln(script, "#!"+shell)
	fmt.Fprintln(script)
	for k, v := range envRead {
		// -r     Make  names  readonly.   These names cannot then be assigned values by subsequent
		//        assignment statements or unset.
		fmt.Fprintf(script, "declare -p %s >/dev/null 2>&1 || declare -r %s=%s\n", k, k, v)
	}
	fmt.Fprintln(script)
	fmt.Fprintln(script, "set -o errexit")
	fmt.Fprintln(script, "set -o errtrace")
	fmt.Fprintln(script, "set -o nounset")
	fmt.Fprintln(script, "set -o pipefail")
	fmt.Fprintln(script, "set -o xtrace")
	fmt.Fprintln(script)
	fmt.Fprintf(script, "# User script for %s\n", cmdName)
	fmt.Fprintln(script)
	fmt.Fprintln(script, code)
	fmt.Fprintln(script)
	fmt.Fprintln(script, "set +o xtrace")
	fmt.Fprintln(script, "set +o pipefail")
	fmt.Fprintln(script, "set +o nounset")
	fmt.Fprintln(script, "set +o errtrace")
	fmt.Fprintln(script, "set +o errexit")
	return
}

func (s *Resetter) signal(verb, param string) {
	io.WriteString(s.stdin, verb)
	io.WriteString(s.stdin, param)
	io.WriteString(s.stdin, "\n")
}

func (s *Resetter) execEach(ctx context.Context, cmd shellCmd) (err error) {
	start := time.Now()

	s.signal("###:exec:", s.scriptsPaths[cmd])
	log.Println("[DBG] sent processing signal to shell singleton:", s.scriptsPaths[cmd])

	select {
	case errcode := <-s.rcoms.errcodes:
		log.Printf("[DBG] shell script %s execution error code: %d", cmd, errcode)
		if errcode != 0 {
			err = fmt.Errorf("script %s exited with non-zero code %d", cmd, errcode)
			log.Println("[ERR]", err)
			return
		}

	case err = <-s.sherr:
		if err != nil {
			log.Printf("[ERR] shell script %s execution error: %s", cmd, err)
			// reason := s.stdboth.String() + "\n" + err.Error()
			// var lines [][]byte
			// for _, line := range strings.Split(reason, "\n") {
			// 	if strings.HasPrefix(line, stdeitherPrefixSkip) {
			// 		continue
			// 	}
			// 	if x := strings.TrimPrefix(line, stdeitherPrefixDropPrefix); x != line {
			// 		lines = append(lines, []byte(x))
			// 	}
			// }
			// err = resetter.NewError(lines)
			return
		}

	case <-ctx.Done():
		if err = ctx.Err(); err != nil {
			log.Printf("[ERR] %s ctx.Done(): %s", cmd, err)
		}
		return

	case <-time.After(scriptTimeout):
		err = context.Canceled
		log.Printf("[ERR] %s scriptTimeout=%s: %s", cmd, scriptTimeout, err)
		return
	}

	log.Printf("[NFO] exec'd %s in %s", cmd, time.Since(start))
	return
}
