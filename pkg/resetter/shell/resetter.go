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

	"github.com/fsnotify/fsnotify"
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
	sherr          chan error
	i, o           string
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

// Terminate cleans up after a resetter.Interface implementation instance
func (s *Resetter) Terminate(ctx context.Context, stdout io.Writer, stderr io.Writer, envRead map[string]string) (err error) {
	if hasStop := strings.TrimSpace(s.Stop) != ""; hasStop {
		if err = s.ExecStop(ctx, stdout, stderr, true, envRead); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
	return
}

type shellCmd int

const (
	cmdGuard shellCmd = iota
	cmdStart
	cmdReset
	cmdStop
)

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
		scriptPrefix := cwid.Prefixed()
		for cmd, command := range map[shellCmd]struct {
			Name, Code string
		}{
			// cmdGuard: {"guard", "exec false"},
			cmdGuard: {"guard", "true"},
			cmdStart: {"start", s.Start},
			cmdReset: {"reset", s.Rst},
			cmdStop:  {"stop", s.Stop},
		} {
			path := fmt.Sprintf("%s_%s.bash", scriptPrefix, command.Name)
			if err = writeScript(path, command.Name, command.Code, envRead); err != nil {
				log.Println("[ERR]", err)
				return
			}
			s.scriptsPaths[cmd] = path
			paths = append(paths, path)
		}

		i := fmt.Sprintf("%s_%s.txt", scriptPrefix, "main_i")
		if err = writeFile(i, nil); err != nil {
			log.Println("[ERR]", err)
		}
		o := fmt.Sprintf("%s_%s.txt", scriptPrefix, "main_o")
		if err = writeFile(o, nil); err != nil {
			log.Println("[ERR]", err)
		}
		main := fmt.Sprintf("%s_%s.bash", scriptPrefix, "main")
		if err = writeMainScript(main, i, o, paths); err != nil {
			return
		}
		s.i, s.o = i, o

		s.sherr = make(chan error, 1)
		go startShell(ctx, s.sherr, stdout, stderr, main)

		// if e := s.execEach(ctx, s.scriptsPaths[cmdGuard]); e == nil || e.Error() != `
		// script failed during Reset:
		// ++ exec false

		// exit status 1
		// `[1:] {
		// 	err = fmt.Errorf("could not setup shell resetter: %s", e)
		// 	log.Println("[ERR]", err)
		// 	return
		// }

		if e := s.execEach(ctx, s.scriptsPaths[cmdGuard]); e != nil {
			err = fmt.Errorf("could not setup shell resetter: %s", e)
			log.Println("[ERR]", err)
			return
		}
	})
	if err != nil {
		return
	}

	for _, cmd := range cmds {
		scriptFile := s.scriptsPaths[cmd]
		if err = s.execEach(ctx, scriptFile); err != nil {
			return
		}
	}
	return
}

func writeFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0640)
}

func writeMainScript(name, i, o string, paths []string) (err error) {
	var script *os.File
	if script, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0740); err != nil {
		return
	}
	defer script.Close()

	mainCode := `
#!/bin/bash -ux

set -o pipefail

trap "echo Shell instance exiting >&2" EXIT

while :; do
	if ! script=$(cat "%s"); then
		# File was deleted
		break
	fi

	if [[ -z "$script" ]]; then
		# No new input yet
		sleep 0.1
		continue
	fi

	source "$script"
	echo $? >"%s"

	while :; do
		if ! script=$(cat "%s"); then
			# File was deleted
			break
		fi
		if [[ -z "$script" ]]; then
			# Input was finally reset
			break
		fi
	done
done
rm -f "%s"
`[1:]
	code := fmt.Sprintf(mainCode, i, o, i, strings.Join(append(paths, i, o, name), `" "`))

	fmt.Fprintln(script, code)
	return
}

func writeScript(scriptFile, cmdName, code string, envRead map[string]string) (err error) {
	var script *os.File
	if script, err = os.OpenFile(scriptFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0740); err != nil {
		return
	}
	defer script.Close()

	fmt.Fprintln(script, "#!/bin/bash")
	fmt.Fprintln(script)
	for k, v := range envRead {
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

// FIXME: this goroutine may leak
func startShell(ctx context.Context, sherr chan error, stdout, stderr io.Writer, mainPath string) {
	var stdboth bytes.Buffer // TODO: mux stderr+stdout and fwd to server to track progress

	exe := exec.CommandContext(ctx, shell, "--norc", "--", mainPath)
	exe.Stdin = nil
	exe.Stdout = io.MultiWriter(&stdboth, stdout)
	exe.Stderr = io.MultiWriter(&stdboth, stderr)

	log.Printf("[DBG] starting shell instance")
	start := time.Now()

	err := exe.Run()

	log.Printf("[NFO] exec'd in %s", time.Since(start))

	for i, line := range bytes.Split(stdboth.Bytes(), []byte{'\n'}) {
		log.Printf("[NFO] STDERR+STDOUT:%d: %q", i, line)
	}

	if err != nil {
		reason := stdboth.String() + "\n" + err.Error()
		err = resetter.NewError(strings.Split(reason, "\n"))
	}

	sherr <- err
}

func (s *Resetter) execEach(ctx context.Context, scriptFile string) (err error) {
	var watcher *fsnotify.Watcher
	if watcher, err = fsnotify.NewWatcher(); err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer watcher.Close()
	if err = watcher.Add(s.o); err != nil {
		log.Println("[ERR]", err)
		return
	}

	start := time.Now()

	mtime := func(f string) time.Time {
		fi, err := os.Stat(f)
		if err != nil {
			panic(err)
		}
		return fi.ModTime()
	}

	before := mtime(s.i)
	log.Println("[NFO] mtime before", before)
	if err = writeFile(s.i, []byte(scriptFile)); err != nil {
		log.Println("[ERR]", err)
		return
	}
	time.Sleep(100 * time.Millisecond)
	after := mtime(s.i)
	log.Println("[NFO] mtime after", after, before == after)
	defer writeFile(s.i, nil)
	log.Println("[DBG] sent processing signal to shell singleton")

	select {
	case err = <-s.sherr:
		log.Println("[ERR] shell script execution error:", err)
		return

	case wevent, ok := <-watcher.Events:
		if !ok {
			err = errors.New("file watcher unexpectedly closed events channel")
			log.Println("[ERR]", err)
			return
		}
		// if true {
		// 	panic(wevent)
		// }
		if !(wevent.Op == fsnotify.Write && wevent.Name == s.o) {
			err = errors.New("file watcher encountered an issue")
			log.Printf("[ERR] op=%s name=%q: %s", wevent.Op, wevent.Name, err)
			return
		}
		log.Println("[DBG] file watcher got the termination signal")

	case werr, ok := <-watcher.Errors:
		if !ok {
			err = errors.New("file watcher unexpectedly closed errors channel")
			log.Println("[ERR]", err)
			return
		}
		err = errors.New("file watcher received unexpected error")
		log.Printf("[ERR] %s: %s", err, werr)
		return

	case <-ctx.Done():
		if err = ctx.Err(); err != nil {
			log.Println("[ERR]", err)
		}
		return

	case <-time.After(scriptTimeout):
		err = context.Canceled
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[NFO] exec'd in %s", time.Since(start))

	var data []byte
	if data, err = os.ReadFile(s.o); err != nil {
		log.Println("[ERR]", err)
		return
	}
	var errcode int
	if errcode, err = strconv.Atoi(strings.TrimSpace(string(data))); err != nil {
		log.Println("[ERR]", err)
		return
	}
	if errcode != 0 {
		err = fmt.Errorf("script exited with non-zero code %d", errcode)
		log.Println("[ERR]", err)
	}
	return
}
