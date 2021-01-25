package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCov(t *testing.T) {
	pathErrCode := os.Getenv("MONKEY_CODEFILE")
	if pathErrCode == "" {
		t.SkipNow()
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	args := strings.Split(os.Getenv("MONKEY_ARGS"), " ")
	os.Args = append([]string{"./" + binName + ".test"}, args...)

	code := actualMain()

	fmt.Println("EXIT", code)
	data := []byte(strconv.Itoa(code))
	err := ioutil.WriteFile(pathErrCode, data, 0644)
	require.NoError(t, err)
}
