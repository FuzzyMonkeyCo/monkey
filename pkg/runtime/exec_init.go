package runtime

import (
	"errors"
	"fmt"
	"os/user"
	"path"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"github.com/chzyer/readline"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func loadDisabled(_th *starlark.Thread, _module string) (starlark.StringDict, error) {
	return nil, errors.New("load() disabled")
}

func initExec() {
	resolve.AllowSet = true            // set([]) (no proto representation)
	resolve.AllowGlobalReassign = true // reassignment to top-level names
	//> Starlark programs cannot be Turing complete
	//> unless the -recursion flag is specified.
	resolve.AllowRecursion = false

	starlark.CompareLimit = 10 // Depth for (Equal|Compare)-ing things

	allow := map[string]struct{}{
		"abs":       {},
		"all":       {},
		"any":       {},
		"bool":      {},
		"bytes":     {},
		"chr":       {},
		"dict":      {},
		"dir":       {},
		"enumerate": {},
		"False":     {},
		"float":     {},
		"getattr":   {},
		"hasattr":   {},
		"hash":      {},
		"int":       {},
		"len":       {},
		"list":      {},
		"max":       {},
		"min":       {},
		"None":      {},
		"ord":       {},
		"print":     {},
		"range":     {},
		"repr":      {},
		"reversed":  {},
		"set":       {},
		"sorted":    {},
		"str":       {},
		"True":      {},
		"tuple":     {},
		"type":      {},
		"zip":       {},
	}
	deny := map[string]struct{}{
		"fail": {},
	}
	starlarktruth.NewModule(starlark.Universe) // Adds `assert that()`
	for f := range starlark.Universe {
		_, allowed := allow[f]
		_, denied := deny[f]
		switch {
		case allowed:
		case denied:
			delete(starlark.Universe, f)
		case f == starlarktruth.Module: // For check tests
		default:
			panic(fmt.Sprintf("unexpected builtin %q", f))
		}
	}
}

// https://github.com/google/starlark-go/blob/bb14e151c28f3b5fc37e053cbe67bd70e9de5de0/doc/spec.md#built-in-methods

var starlarkExtendedUniverse = map[string][]string{
	"bytes": {
		"elems",
	},

	"dict": {
		"clear",
		"get",
		"items",
		"keys",
		"pop",
		"popitem",
		"setdefault",
		"update",
		"values",
	},

	"list": {
		"append",
		"clear",
		"extend",
		"index",
		"insert",
		"pop",
		"remove",
	},

	"string": {
		"capitalize",
		"codepoint_ords",
		"codepoints",
		"count",
		"elem_ords",
		"elems",
		"endswith",
		"find",
		"format",
		"index",
		"isalnum",
		"isalpha",
		"isdigit",
		"islower",
		"isspace",
		"istitle",
		"isupper",
		"join",
		"lower",
		"lstrip",
		"partition",
		"replace",
		"rfind",
		"rindex",
		"rpartition",
		"rsplit",
		"rstrip",
		"split",
		"splitlines",
		"startswith",
		"strip",
		"title",
		"upper",
	},

	"set": {
		"union",
	},
}

const (
	replPrompt    = ">>> "
	replPromptSub = "... "
)

func newREPLConfig() (*readline.Config, error) {
	whoami, err := user.Current()
	if err != nil {
		return nil, err
	}

	// TODO: completer for methods of types + taylored for Python (not for a CLI)
	// (use starlarkExtendedUniverse)

	// TODO:
	// >>> def blop(x):
	// ...     retur n x
	// <stdin>:2:12: got identifier, want newline
	// =>
	// >>> def blop(x):
	// ...     retur n x
	//               ^
	// got identifier, want newline

	prefixes := make([]readline.PrefixCompleterInterface, 0, len(starlark.Universe))
	prefixes = append(prefixes, readline.PcItem("assert"))
	for p := range starlark.Universe {
		if p == "assert" {
			continue
		}
		item := readline.PcItem(p + "(")
		if p == "True" || p == "False" {
			item = readline.PcItem(p)
		}
		prefixes = append(prefixes, item)
	}

	cfg := &readline.Config{
		Prompt:              replPrompt,
		HistoryFile:         path.Join(whoami.HomeDir, ".fuzzymonkey_starlark_history"),
		HistorySearchFold:   true,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		FuncFilterInputRune: filterInput,
		AutoComplete:        readline.NewPrefixCompleter(prefixes...),
	}
	return cfg, nil
}
