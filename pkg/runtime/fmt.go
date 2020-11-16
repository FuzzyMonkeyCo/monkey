package runtime

import (
	"fmt"
	"log"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/convertast"
	"go.starlark.net/syntax"
)

// Format standardizes Starlark codes
func Format(W bool) bool {
	ast, err := syntax.Parse(localCfg, nil, syntax.RetainComments)
	if err != nil {
		log.Println("[ERR]", err)
		return false
	}
	newAst := convertast.ConvFile(ast)
	if !W {
		fmt.Print(build.FormatString(newAst))
	}
	return true
}
