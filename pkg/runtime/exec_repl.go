package runtime

// Inspired from https://github.com/google/starlark-go/blob/70c0e40ae1287fd2c0aa43184b482838d8db051d/repl/repl.go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/chzyer/readline"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
)

// JustExecREPL executes a Starlark Read-Eval-Print Loop
func (rt *Runtime) JustExecREPL(ctx context.Context) error {
	fmt.Println("# Welcome to Starlark! Learn about the language at https://FIXME")

	fmt.Println(`# To express assertions, use "assert":`)
	fmt.Printf(strings.Repeat(" ", len(replPrompt)))
	replPrint("assert that(x != 42).is_truthy()")

	fmt.Println("# or better yet, the more expressive:")
	fmt.Printf(strings.Repeat(" ", len(replPrompt)))
	replPrint("assert that(x).is_not_equal_to(42)")

	rt.thread.Name = "REPL"
	rt.thread.Load = loadDisabled
	return repl(ctx, rt.thread, rt.globals)
}

func repl(ctx context.Context, thread *starlark.Thread, globals starlark.StringDict) error {
	cfg, err := newREPLConfig()
	if err != nil {
		return err
	}
	rl, err := readline.NewEx(cfg)
	if err != nil {
		return err
	}
	defer rl.Close()

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt)
	defer signal.Stop(interrupted)

	prevBadExpr := false
	for {
		badExpr, err := rep(ctx, interrupted, rl, thread, globals)
		switch err {
		case readline.ErrInterrupt:
			fmt.Println(err)
			continue
		case io.EOF:
			if err := starlarktruth.Close(thread); err != nil {
				return starTrickError(err)
			}
			err = nil
			if prevBadExpr {
				err = errors.New("") // Signal last expr failed for non-zero exit code
			}
			return err
		}
		prevBadExpr = badExpr
	}
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

// rep reads, evaluates, and prints one item.
//
// It returns an error (possibly readline.ErrInterrupt)
// only if readline failed. Starlark errors are printed.
func rep(
	ctx context.Context,
	interrupted chan os.Signal,
	rl *readline.Instance,
	thread *starlark.Thread,
	globals starlark.StringDict,
) (bool, error) {
	// Each item gets its own context,
	// which is cancelled by a SIGINT.
	//
	// Note: during Readline calls, Control-C causes Readline to return
	// ErrInterrupt but does not generate a SIGINT.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-interrupted:
			cancel()
		case <-ctx.Done():
		}
	}()

	eof := false

	// readline returns EOF, ErrInterrupted, or a line including "\n".
	rl.SetPrompt(replPrompt)
	readline := func() ([]byte, error) {
		line, err := rl.Readline()
		line = string(starTrick([]byte(line)))
		rl.SetPrompt(replPromptSub)
		if err != nil {
			if err == io.EOF {
				eof = true
			}
			return nil, err
		}
		return []byte(line + "\n"), nil
	}

	// parse
	f, err := syntax.ParseCompoundStmt(replPromptFile, readline)
	if err != nil {
		if eof {
			return false, io.EOF
		}
		replError(err)
		return false, nil
	}

	if expr := soleExpr(f); expr != nil {
		// eval
		v, err := starlark.EvalExpr(thread, expr, globals)
		if err != nil {
			replError(err)
			return true, nil
		}

		// print
		if v != starlark.None {
			replPrint(v.String())
		}
	} else if err := starlark.ExecREPLChunk(f, thread, globals); err != nil {
		replError(err)
		return true, nil
	}

	return false, nil
}

func soleExpr(f *syntax.File) syntax.Expr {
	if len(f.Stmts) == 1 {
		if stmt, ok := f.Stmts[0].(*syntax.ExprStmt); ok {
			return stmt.X
		}
	}
	return nil
}

var (
	replPPlexer = chroma.Coalesce(lexers.Get("python"))
	replPPfmter = formatters.TTY16m
	replPPstyle = styles.Monokai
)

func replPrint(vs string) {
	if strings.HasPrefix(vs, "<") { // "<built-in ...
		fmt.Println(vs)
		return
	}

	it, err := replPPlexer.Tokenise(nil, vs)
	if err != nil {
		panic(err)
	}
	if err := replPPfmter.Format(os.Stdout, replPPstyle, it); err != nil {
		panic(err)
	}
	fmt.Println()
}

func replError(err error) {
	w := os.Stderr

	pinner := func(msg string, col int32) bool {
		// Starlark syntax.Position:
		//   1-based column (rune) number; 0 if column unknown
		if col > 0 {
			if col == 1 {
				col++
			}
			// TODO: place pin at beginning instead of end of token
			// >>> 11111111111        999999999999999
			//                                      ^
			// invalid syntax
			//=>
			// >>> 11111111111        999999999999999
			//                        ^
			// invalid syntax
			ws := strings.Repeat(" ", len(replPrompt)+int(col-1)-len("^"))
			as.ColorERR.Fprintf(w, "%s^\n", ws)
			as.ColorERR.Fprintln(w, msg)
			return true
		}
		return false
	}

	err = starTrickError(err)

	switch err := err.(type) {
	case syntax.Error:
		if pinner(err.Msg, err.Pos.Col) {
			return
		}
	case resolve.Error:
		if pinner(err.Msg, err.Pos.Col) {
			return
		}
	case resolve.ErrorList:
		if pinner(err[0].Msg, err[0].Pos.Col) {
			return
		}
	}

	as.ColorERR.Fprintln(w, err)
}
