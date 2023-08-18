package shell

import (
	"errors"
	"log"
)

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
		hasStart = "" != s.Start
		hasReset = "" != s.Rst
		hasStop  = "" != s.Stop
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
