package main

import (
	"fmt"
	"os"
)

type doneCmd struct {
	V       uint   `json:"v"`
	Cmd     string `json:"cmd"`
	Failure bool   `json:"failure"`
}

func (cmd doneCmd) Kind() string {
	return cmd.Cmd
}

func (cmd doneCmd) Exec(cfg *ymlCfg) []byte {
	return nil
}

func testOutcome(cmd doneCmd) int {
	if cmd.Failure {
		os.Stdout.Write([]byte{'\n'})
		fmt.Println("A bug was detected and minified!\n")
		return 6
	}
	return 0
}
