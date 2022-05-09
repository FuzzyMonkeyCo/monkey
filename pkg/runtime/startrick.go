package runtime

// The Starlark Trick is to allow using `assert that(x).y(z)`
// when Starlark does not treat `assert` as a statement (a la Python)
// but as a value: `assert.that(x).y(z)`

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
)

func starTrickError(err error) error {
	trick := func(data string) string {
		return strings.Replace(data, ": assert.that(", ": assert that(", 1)
	}
	switch e := err.(type) {
	case *starlark.EvalError:
		return errors.New(trick(e.Backtrace())) // No way to build an error of same type
	case *starlarktruth.UnresolvedError:
		const prefix = "Traceback (most recent call last):\n  "
		return &starlarktruth.UnresolvedError{Msg: prefix + trick(e.Error())}
	default:
		return err
	}
}

var startrick = regexp.MustCompile(`(^|\n|[^"']+)\s*assert\s+that\s*`)

func starTrick(data []byte) []byte {
	return startrick.ReplaceAllFunc(data, starTrickFunc)
}

func starTrickPerLine(data []byte) {
	if bytes.ContainsRune(data, '#') {
		return
	}

	fixes := []string{" ", "\t"}
	for _, prefix := range fixes {
		for _, suffix := range fixes {
			if bytes.HasPrefix(data, []byte("assert"+suffix)) {
				data[len("assert"+suffix)-1] = '.'
			}

			if k := bytes.Index(data, []byte(prefix+"assert"+suffix)); k > -1 {
				data[k+len(prefix+"assert"+suffix)-1] = '.'
			}
		}
	}
}

func starTrickFunc(data []byte) []byte {
	for i := 0; ; {
		n := bytes.IndexAny(data[i:], "\n\r")
		if n < 0 && len(data[i:]) > 0 {
			starTrickPerLine(data[i:])
			break
		}
		starTrickPerLine(data[i : i+n])
		i += n + 1
	}
	return data
}

var startrickdual = regexp.MustCompile(`(^|\n|[^"']+)\s*assert[.]that[(]`)

func starTrickDual(data []byte) []byte {
	return startrickdual.ReplaceAllFunc(data, starTrickDualFunc)
}

func starTrickDualFunc(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("assert.that("), []byte("assert that("))
}
