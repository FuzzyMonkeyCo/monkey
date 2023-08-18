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
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
)

const (
	stdeitherPrefixSkip       = "+ "
	stdeitherPrefixDropPrefix = "++ "
	comPrefix                 = "###:" //FIXME: pick from random alnum at first run
	comExec                   = comPrefix + "exec:"
	comExit                   = comPrefix + "exit:"
	comExitcode               = comPrefix + "exitcode:"
	comUnexpected             = comPrefix + "unexpected:"
)

func (s *Resetter) exec(ctx context.Context, shower progresser.Shower, envRead map[string]string, cmds ...shellCmd) (err error) {
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

		// s.rcoms = &rcoms{errcodes: make(chan uint8)}
		s.rcoms = make(chan uint8)
		coms := newlinesWriter(func(data []byte) {
			if n := len(data); n > 0 {
				if x := bytes.TrimPrefix(data, []byte(comExitcode)); n != len(x) {
					if y, err := strconv.ParseInt(string(x), 10, 8); err == nil {
						s.rcoms <- uint8(y)
					}
				}
				if x := bytes.TrimPrefix(data, []byte(comUnexpected)); n != len(x) {
					log.Printf("[ERR] unexpected rcoms: %s", x)
				}
			}
		})

		stdout := newlinesWriter(func(data []byte) {
			if n := len(data); n > 0 {
				if bytes.HasPrefix(data, []byte("+ ")) {
					return
				}
				if x := bytes.TrimPrefix(data, []byte("++ ")); n != len(x) {
					if string(x) != "set +o xtrace" {
						shower.Printf(" ❯ %s", x)
					}
					return
				}
				if x := bytes.TrimPrefix(data, []byte(comExitcode)); n != len(x) {
					return
				}
				shower.Printf("%s", data)
			}
		})

		stderr := newlinesWriter(func(data []byte) {
			if n := len(data); n > 0 {
				if bytes.HasPrefix(data, []byte("+ ")) {
					return
				}
				if x := bytes.TrimPrefix(data, []byte("++ ")); n != len(x) {
					if string(x) != "set +o xtrace" {
						shower.Errorf(" ❯ %s", x)
					}
					return
				}
				if x := bytes.TrimPrefix(data, []byte(comExitcode)); n != len(x) {
					return
				}
				shower.Errorf("%s", data)
			}
		})

		// TODO: mux stderr+stdout and fwd to server to track progress
		exe.Stdout = io.MultiWriter(stdout, coms)
		exe.Stderr = io.MultiWriter(stderr, coms)

		go func() {
			log.Printf("[DBG] starting shell instance")
			defer log.Println("[NFO] shell singleton exited")

			if err := exe.Run(); err != nil {
				log.Println("[ERR] forwarding error:", err)
				lines := append([][]byte(nil), []byte(err.Error()))
				s.sherr <- resetter.NewError(lines)
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
	'`+comExec+`'*) x=${x:9} ;;
	'`+comExit+`') exit 0 ;;
	*) echo "`+comUnexpected+`$x" && exit 42 ;;
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
	echo "`+comExitcode+`$?"
done
`)
	return
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

	s.signal(comExec, s.scriptsPaths[cmd])
	log.Println("[DBG] sent processing signal to shell singleton:", s.scriptsPaths[cmd])

	select {
	case errcode := <-s.rcoms:
		log.Printf("[DBG] shell script %s execution error code: %d", cmd, errcode)
		if errcode != 0 {
			err = fmt.Errorf("script %s exited with non-zero code %d", cmd, errcode)
			log.Println("[ERR]", err)
			return
		}

	case err = <-s.sherr:
		if err != nil {
			log.Printf("[ERR] shell script %s execution error: %s", cmd, err)
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
