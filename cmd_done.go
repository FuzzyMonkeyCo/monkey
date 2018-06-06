package main

import (
	"fmt"
	"os"
)

type doneCmd struct {
	V       uint    `json:"v"`
	Cmd     cmdKind `json:"cmd"`
	Failure bool    `json:"failure"`
}

func (cmd *doneCmd) Kind() cmdKind {
	return cmd.Cmd
}

func (cmd *doneCmd) Exec(cfg *YmlCfg) ([]byte, error) {
	return nil, nil
}

func fuzzOutcome(cmd *doneCmd) int {
	os.Stdout.Write([]byte{'\n'})
	fmt.Printf("Ran %d tests totalling %d requests\n", lastLane.T, totalR)

	if cmd.Failure {
		d, m := shrinkingFrom.T, lastLane.T-shrinkingFrom.T
		if m != 1 {
			fmt.Printf("A bug was detected after %d tests then shrunk %d times!\n", d, m)
		} else {
			fmt.Printf("A bug was detected after %d tests then shrunk once!\n", d)
		}
		return 6
	}
	fmt.Println("No bugs found... yet.")
	return 0
}
