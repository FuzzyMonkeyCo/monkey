package main

import (
	"os"
	"strings"
	"testing"
)

func TestCov(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	args := strings.Split(os.Getenv("MONKEY_ARGS"), " ")
	os.Args = append([]string{"./" + binName + ".test"}, args...)

	actualMain()
}
