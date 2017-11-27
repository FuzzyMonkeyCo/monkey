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

func (cmd *doneCmd) Kind() string {
	return cmd.Cmd
}

func (cmd *doneCmd) Exec(cfg *ymlCfg) ([]byte, error) {
	return nil, nil
}

func testOutcome(cmd *doneCmd) int {
	os.Stdout.Write([]byte{'\n'})
	if cmd.Failure {
		fmt.Println("A bug was detected and minified!")
		return 6
	}
	fmt.Println("No bugs found yet!")
	return 0
}
