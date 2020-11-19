package runtime

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/bazelbuild/buildtools/build"
	"go.starlark.net/syntax"
)

// Format standardizes Starlark codes
func Format(W bool) bool {
	ast, err := syntax.Parse(localCfg, nil, syntax.RetainComments)
	if err != nil {
		log.Println("[ERR]", err)
		return false
	}
	newAst := ConvFile(ast)
	str := build.FormatString(newAst)

	if !W {
		fmt.Print(str)
		return true
	}

	if err := ioutil.WriteFile(localCfg, []byte(str), 0644); err != nil {
		log.Println("[ERR]", err)
		return false
	}

	return true
}
