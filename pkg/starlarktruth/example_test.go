package starlarktruth_test

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"go.starlark.net/starlark"
)

func Example() {
	starlarktruth.Module = "should"
	starlarktruth.Method = "verify"

	starlarktruth.NewModule(starlark.Universe)

	data := `
squares = [x*x for x in range(10)]
should.verify(squares).has_size(10)

print(greeting + ", world")
`[1:]

	th := &starlark.Thread{
		Name:  "truth",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	_, err := starlark.ExecFile(th, "some/file.star", data, starlark.StringDict{
		"greeting": starlark.String("hello"),
	})
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			panic(evalErr.Backtrace())
		}
		panic(err)
	}

	// Fail if any subject wasn't applied a property.
	if err := starlarktruth.Close(th); err != nil {
		panic(err)
	}

	// Output:
	// hello, world
}
