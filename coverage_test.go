package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestCov(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	args := strings.Split(os.Getenv("MONKEY_ARGS"), " ")
	os.Args = append([]string{"./" + binName + ".test"}, args...)

	code := actualMain()

	fmt.Println("EXIT", code)
	data := []byte(strconv.Itoa(code))
	path := os.Getenv("MONKEY_CODEFILE")
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}
