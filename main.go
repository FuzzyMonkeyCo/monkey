package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/code"
	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	monkey "github.com/FuzzyMonkeyCo/monkey/pkg/runtime"
	"github.com/FuzzyMonkeyCo/monkey/pkg/update"
	docopt "github.com/docopt/docopt-go"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/mapstructure"
)

//go:generate echo Let's go bananas!

const (
	binName    = "monkey"
	binSHA     = "feedb065"
	binVersion = "0.0.0"
	githubSlug = "FuzzyMonkeyCo/" + binName

	// Environment variables used
	envAPIKey = "FUZZYMONKEY_API_KEY"
)

var (
	clientUtils = &http.Client{
		Timeout: 10 * time.Second,
	}
)

func main() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)
	os.Exit(actualMain())
}

type params struct {
	Fuzz, Shrink                   bool
	Lint, Schema                   bool
	Init, Env, Login, Logs         bool
	Exec, Start, Reset, Stop, Repl bool
	Update                         bool     `mapstructure:"--update"`
	ShowSpec                       bool     `mapstructure:"--show-spec"`
	N                              uint32   `mapstructure:"--tests"`
	Verbosity                      uint8    `mapstructure:"-v"`
	LogOffset                      uint64   `mapstructure:"--previous"`
	ValidateAgainst                string   `mapstructure:"--validate-against"`
	EnvVars                        []string `mapstructure:"VAR"`
}

func usage(binTitle string) (args *params, ret int) {
	B := as.ColorNFO.Sprintf(binName)
	usage := binTitle + `

Usage:
  ` + B + ` [-vvv] init [--with-magic]
  ` + B + ` [-vvv] login [--user=USER]
  ` + B + ` [-vvv] fuzz [--tests=N] [--seed=SEED] [--tag=TAG]...
                     [--only=REGEX]... [--except=REGEX]...
                     [--calls-with-input=SCHEMA]... [--calls-without-input=SCHEMA]...
                     [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  ` + B + ` [-vvv] shrink --test=ID [--seed=SEED] [--tag=TAG]...
  ` + B + ` [-vvv] lint [--show-spec]
  ` + B + ` [-vvv] schema [--validate-against=REF]
  ` + B + ` [-vvv] exec (repl | start | reset | stop)
  ` + B + ` [-vvv] -h | --help
  ` + B + ` [-vvv]      --update
  ` + B + ` [-vvv] -V | --version
  ` + B + ` [-vvv] env [VAR ...]
  ` + B + ` logs [--previous=N]

Options:
  -v, -vv, -vvv                  Debug verbosity level
  -h, --help                     Show this screen
  -U, --update                   Ensures ` + B + ` is current
  -V, --version                  Show version
  --seed=SEED                    Use specific parameters for the RNG
  --validate-against=REF         Schema $ref to validate STDIN against
  --tag=TAG                      Labels that can help classification
  --test=ID                      Which test to shrink
  --tests=N                      Number of tests to run [default: 100]
  --only=REGEX                   Only test matching calls
  --except=REGEX                 Do not test these calls
  --calls-with-input=SCHEMA      Test calls which can take schema PTR as input
  --calls-without-output=SCHEMA  Test calls which never output schema PTR
  --user=USER                    Authenticate on fuzzymonkey.co as USER
  --with-magic                   Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  ` + B + ` --update
  ` + B + ` exec reset
  ` + B + ` fuzz --only /pets --calls-without-input=NewPet --tests=0
  echo '"kitty"' | ` + B + ` schema --validate-against=#/components/schemas/PetKind`

	// https://github.com/docopt/docopt.go/issues/59
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		// Usage shown: bad args
		as.ColorERR.Println(err)
		ret = code.Failed
		return
	}

	if opts["--version"].(bool) {
		fmt.Println(binTitle)
		ret = code.OK
		return
	}

	args = &params{}
	if err := mapstructure.WeakDecode(opts, args); err != nil {
		as.ColorERR.Println(err)
		return nil, code.Failed
	}
	return
}

func actualMain() int {
	binTitle := strings.Join([]string{binName, binVersion, binSHA,
		runtime.Version(), runtime.GOARCH, runtime.GOOS}, "\t")
	args, ret := usage(binTitle)
	if args == nil {
		return ret
	}

	if args.Logs {
		offset := args.LogOffset
		if offset == 0 {
			offset = 1
		}
		return doLogs(offset)
	}

	if err := cwid.MakePwdID(binName, 0); err != nil {
		return retryOrReport()
	}
	logCatchall, err := os.OpenFile(cwid.LogFile(), os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		as.ColorERR.Println(err)
		return retryOrReport()
	}
	defer logCatchall.Close()
	logFiltered := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DBG", "NFO", "ERR", "NOP"},
		MinLevel: logLevel(args.Verbosity),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Printf("[ERR] (not an error) %s %s %#v\n", binTitle, cwid.LogFile(), args)

	if args.Init || args.Login {
		// FIXME: implement init & login
		as.ColorERR.Println("Action not implemented yet")
		return code.Failed
	}

	if args.Update {
		return doUpdate()
	}

	if args.Env {
		return doEnv(args.EnvVars)
	}

	rt, err := monkey.NewMonkey(binTitle)
	if err != nil {
		as.ColorERR.Println(err)
		return code.Failed
	}
	// Always lint
	if err := rt.Lint(args.ShowSpec); err != nil {
		as.ColorERR.Println(err)
		return code.FailedLint
	}
	if args.Lint {
		e := "Configuration is valid."
		log.Println("[NFO]", e)
		as.ColorNFO.Println(e)
		return code.OK
	}

	if args.Exec {
		var fn func() error
		switch {
		case args.Start:
			fn = rt.JustExecStart
		case args.Reset:
			fn = rt.JustExecReset
		case args.Stop:
			fn = rt.JustExecStop
		default:
			fn = rt.JustExecREPL
		}
		if err := fn(); err != nil {
			as.ColorERR.Println(err)
			return code.FailedExec
		}
		return code.OK
	}

	if args.Schema {
		ref := args.ValidateAgainst
		if ref == "" {
			rt.WriteAbsoluteReferences(os.Stdout)
			return code.OK
		}

		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Println("[ERR]", err)
			return code.FailedSchema
		}

		if err := rt.ValidateAgainstSchema(ref, data); err != nil {
			switch err {
			case modeler.ErrUnparsablePayload:
			case modeler.ErrNoSuchSchema:
				as.ColorERR.Printf("No such $ref '%s'\n", ref)
				rt.WriteAbsoluteReferences(os.Stdout)
			default:
				as.ColorERR.Println(err)
			}
			return code.FailedSchema
		}
		as.ColorNFO.Println("Payload is valid")
		return code.OK
	}

	var apiKey string
	if apiKey = os.Getenv(envAPIKey); apiKey == "" {
		err := fmt.Errorf("$%s is unset", envAPIKey)
		log.Println("[ERR]", err)
		as.ColorERR.Println(err)
		return code.Failed
	}

	as.ColorNFO.Printf("%d named schemas\n", rt.InputsCount())
	if err = rt.FilterEndpoints(os.Args); err != nil {
		as.ColorERR.Println(err)
		return code.Failed
	}

	rt.Ntensity = args.N
	if args.N == 0 {
		as.ColorERR.Println("No tests to run.")
		return code.Failed
	}
	as.ColorNFO.Printf("\n Running tests...\n\n")

	ctx := context.Background()
	closer, err := rt.Dial(ctx, binTitle, apiKey)
	if err != nil {
		return retryOrReport()
	}
	defer closer()

	if err := rt.Fuzz(ctx); err != nil {
		return retryOrReportThenCleanup(err)
	}
	fmt.Println()
	fmt.Println()
	// if rt.TestsSucceeded() {
	return code.OK
	// }
	// return code.FailedFuzz
}

func logLevel(verbosity uint8) logutils.LogLevel {
	lvl := map[uint8]string{
		0: "NOP",
		1: "ERR",
		2: "NFO",
		3: "DBG",
	}[verbosity]
	return logutils.LogLevel(lvl)
}

func doLogs(offset uint64) int {
	if err := cwid.MakePwdID(binName, offset); err != nil {
		return retryOrReport()
	}

	fn := cwid.LogFile()
	os.Stderr.WriteString(fn + "\n")
	f, err := os.Open(fn)
	if err != nil {
		as.ColorERR.Println(err)
		return code.Failed
	}
	defer f.Close()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		as.ColorERR.Println(err)
		return retryOrReport()
	}
	return code.OK
}

func doUpdate() int {
	rel := &update.GithubRelease{
		Slug:   githubSlug,
		Name:   binName,
		Client: clientUtils,
	}
	latest, err := rel.PeekLatestRelease()
	if err != nil {
		return retryOrReport()
	}

	if latest != binVersion {
		fmt.Println("A version newer than", binVersion, "is out:", latest)
		if err := rel.ReplaceCurrentRelease(latest); err != nil {
			fmt.Println("The update failed 🙈 please try again")
			return code.FailedUpdate
		}
	}
	return code.OK
}

func doEnv(vars []string) int {
	all := map[string]bool{
		envAPIKey: false,
	}
	penv := func(key string) { fmt.Printf("%s=%q\n", key, os.Getenv(key)) }
	if len(vars) == 0 {
		for key := range all {
			penv(key)
		}
		return code.OK
	}

	for _, key := range vars {
		if printed, ok := all[key]; !ok || printed {
			return code.Failed
		}
		all[key] = true
		penv(key)
	}
	return code.OK
}

func retryOrReportThenCleanup(err error) int {
	defer as.ColorWRN.Println("You might want to run $", binName, "exec stop")
	if _, ok := err.(*resetter.Error); ok {
		as.ColorERR.Println(err)
		return code.FailedExec
	}
	return retryOrReport()
}

func retryOrReport() int {
	const issues = "https://github.com/" + githubSlug + "/issues"
	const email = "ook@fuzzymonkey.co"
	w := os.Stderr
	fmt.Fprintln(w, "\nLooks like something went wrong... Maybe try again with -v?")
	fmt.Fprintf(w, "\nYou may want to run `monkey --update`.\n")
	fmt.Fprintf(w, "\nIf that doesn't fix it, take a look at %s\n", cwid.LogFile())
	fmt.Fprintf(w, "or come by %s\n", issues)
	fmt.Fprintf(w, "or drop us a line at %s\n", email)
	fmt.Fprintln(w, "\nThank you for your patience & sorry about this :)")
	return code.Failed
}
