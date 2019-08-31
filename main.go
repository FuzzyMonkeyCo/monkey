package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/lib"
	"github.com/docopt/docopt-go"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/mapstructure"
)

//go:generate echo Let's go bananas!

const (
	/// CLI statuses
	statusOK     = 0
	statusFailed = 1
	// Something happened during linting
	statusFailedLint = 2
	// `binName` executable could not be upgraded
	statusFailedUpdate = 3
	// Some external dependency is missing (probably bash)
	statusFailedRequire = 5
	// Fuzzing found a bug!
	statusFailedFuzz = 6
	// A user command (start, reset, stop) failed
	statusFailedExec = 7
	// Validating payload against schema failed
	statusFailedSchema = 9

	binName    = "monkey"
	binSHA     = "feedb065"
	binVersion = "0.0.0"
	githubSlug = "FuzzyMonkeyCo/" + binName
	wsURL      = "ws://api.dev.fuzzymonkey.co:7077/1/fuzz"

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
	HideConfig                     bool     `mapstructure:"--hide-config"`
	ShowSpec                       bool     `mapstructure:"--show-spec"`
	N                              uint32   `mapstructure:"--tests"`
	Verbosity                      uint8    `mapstructure:"-v"`
	LogOffset                      uint64   `mapstructure:"--previous"`
	ValidateAgainst                string   `mapstructure:"--validate-against"`
	EnvVars                        []string `mapstructure:"VAR"`
}

func usage(binTitle string) (args *params, ret int) {
	B := lib.ColorNFO.Sprintf(binName)
	usage := binTitle + `

Usage:
  ` + B + ` [-vvv] init [--with-magic]
  ` + B + ` [-vvv] login [--user=USER]
  ` + B + ` [-vvv] fuzz [--tests=N] [--seed=SEED] [--tag=TAG]...
                     [--only=REGEX]... [--except=REGEX]...
                     [--calls-with-input=SCHEMA]... [--calls-without-input=SCHEMA]...
                     [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  ` + B + ` [-vvv] shrink --test=ID [--seed=SEED] [--tag=TAG]...
  ` + B + ` [-vvv] lint [--show-spec] [--hide-config]
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
  --hide-config                  Do not show YAML configuration while linting
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
  ` + B + ` fuzz --only /pets --calls-without-input=NewPet --tests=0
  echo '"kitty"' | ` + B + ` schema --validate-against=#/components/schemas/PetKind`

	// https://github.com/docopt/docopt.go/issues/59
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		// Usage shown: bad args
		lib.ColorERR.Println(err)
		ret = statusFailed
		return
	}

	if opts["--version"].(bool) {
		fmt.Println(binTitle)
		return // ret = statusOK
	}

	args = &params{}
	if err := mapstructure.WeakDecode(opts, args); err != nil {
		lib.ColorERR.Println(err)
		return nil, statusFailed
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

	if err := lib.MakePwdID(binName, 0); err != nil {
		return retryOrReport()
	}
	logCatchall, err := os.OpenFile(lib.LogID(), os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		lib.ColorERR.Println(err)
		return retryOrReport()
	}
	defer logCatchall.Close()
	logFiltered := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DBG", "NFO", "ERR", "NOP"},
		MinLevel: logLevel(args.Verbosity),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Printf("[ERR] (not an error) %s %s %#v\n", binTitle, lib.LogID(), args)

	if args.Init || args.Login {
		// FIXME: implement init & login
		lib.ColorERR.Println("Action not implemented yet")
		return statusFailed
	}

	if args.Update {
		return doUpdate()
	}

	if args.Env {
		return doEnv(args.EnvVars)
	}

	cfg, err := lib.NewCfg(args.Lint && !args.HideConfig)
	if err != nil {
		lib.ColorERR.Println(err)
		return statusFailed
	}
	if args.Lint {
		e := fmt.Sprintf("%s is a valid v%d configuration", lib.LocalCfg, cfg.Version)
		log.Println("[NFO]", e)
		lib.ColorNFO.Println(e)
	}

	if args.Exec {
		switch {
		case args.Repl:
			if err := lib.DoExecREPL(); err != nil {
				lib.ColorERR.Println(err)
				return statusFailed
			}
			return statusOK
		case args.Start:
			return doExec(cfg, lib.ExecKind_start)
		case args.Reset:
			return doExec(cfg, lib.ExecKind_reset)
		case args.Stop:
			return doExec(cfg, lib.ExecKind_stop)
		default:
			return retryOrReport()
		}
	}

	docPath, blob, err := cfg.FindThenReadBlob()
	if err != nil {
		return retryOrReport()
	}

	// Always lint before fuzzing
	vald, err := lib.DoLint(docPath, blob, args.ShowSpec)
	if err != nil {
		return statusFailedLint
	}
	if args.Lint {
		err := fmt.Errorf("%s is a valid %v specification", docPath, cfg.Kind)
		log.Println("[NFO]", err)
		lib.ColorNFO.Println(err)
		return statusOK
	}

	if args.Schema {
		return doSchema(vald, args.ValidateAgainst)
	}

	if cfg.ApiKey = os.Getenv(envAPIKey); cfg.ApiKey == "" {
		err = fmt.Errorf("$%s is unset", envAPIKey)
		log.Println("[ERR]", err)
		lib.ColorERR.Println(err)
		return statusFailed
	}

	lib.ColorNFO.Printf("%d named schemas\n", len(vald.Refs))
	eids, err := vald.FilterEndpoints(os.Args)
	if err != nil {
		lib.ColorERR.Println(err)
		return statusFailed
	}
	cfg.EIDs = eids
	cfg.N = args.N
	if cfg.N == 0 {
		lib.ColorERR.Println("No tests to run.")
		return statusFailed
	}
	mnk := lib.NewMonkey(cfg, vald, binTitle)
	lib.ColorNFO.Printf("\n Running %d tests...\n\n", cfg.N)
	return doFuzz(mnk)
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		log.Println("[ERR]", err)
		lib.ColorERR.Println(err)
		panic(err)
	}
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
	if err := lib.MakePwdID(binName, offset); err != nil {
		return retryOrReport()
	}

	fn := lib.LogID()
	os.Stderr.WriteString(fn + "\n")
	f, err := os.Open(fn)
	if err != nil {
		lib.ColorERR.Println(err)
		return statusFailed
	}
	defer f.Close()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		lib.ColorERR.Println(err)
		return retryOrReport()
	}
	return statusOK
}

func doUpdate() int {
	rel := &lib.GithubRelease{
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
			return statusFailedUpdate
		}
	}
	return statusOK
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
		return statusOK
	}

	for _, key := range vars {
		if printed, ok := all[key]; !ok || printed {
			return statusFailed
		}
		all[key] = true
		penv(key)
	}
	return statusOK
}

func doSchema(vald *lib.Validator, ref string) int {
	refs := vald.Refs
	refsCount := len(refs)
	if ref == "" {
		log.Printf("[NFO] found %d refs\n", refsCount)
		lib.ColorNFO.Printf("Found %d refs\n", refsCount)
		vald.WriteAbsoluteReferences(os.Stdout)
		return statusOK
	}

	if err := vald.ValidateAgainstSchema(ref); err != nil {
		switch err {
		case lib.ErrInvalidPayload:
		case lib.ErrNoSuchRef:
			lib.ColorERR.Printf("No such $ref '%s'\n", ref)
			if refsCount > 0 {
				fmt.Println("Try one of:")
				vald.WriteAbsoluteReferences(os.Stdout)
			}
		default:
			lib.ColorERR.Println(err)
		}
		return statusFailedSchema
	}
	lib.ColorNFO.Println("Payload is valid")
	return statusOK
}

func doExec(cfg *lib.UserCfg, kind lib.ExecKind) int {
	if _, err := os.Stat(lib.Shell()); os.IsNotExist(err) {
		log.Println(lib.Shell(), "is required")
		return statusFailedRequire
	}
	if err := lib.SnapEnv(lib.EnvID()); err != nil {
		return retryOrReport()
	}
	defer ensureDeleted(lib.EnvID())

	act, err := lib.ExecuteScript(cfg, kind)
	if err != nil {
		lib.ColorERR.Println(err)
	}
	if err != nil || act.Failure || !act.Success {
		return statusFailedExec
	}
	return statusOK
}

func doFuzz(mnk *lib.Monkey) int {
	if _, err := os.Stat(lib.Shell()); os.IsNotExist(err) {
		log.Println("[ERR]", lib.Shell(), "is required")
		return statusFailedRequire
	}

	if err := lib.SnapEnv(lib.EnvID()); err != nil {
		return retryOrReport()
	}
	defer ensureDeleted(lib.EnvID())

	if err := mnk.Dial(wsURL); err != nil {
		return retryOrReport()
	}

	if err := mnk.FuzzingLoop(lib.Action(&lib.DoFuzz{})); err != nil {
		return retryOrReportThenCleanup(err)
	}
	if mnk.TestsSucceeded() {
		return statusOK
	}
	return statusFailedFuzz
}

func retryOrReportThenCleanup(err error) int {
	defer lib.ColorWRN.Println("You might want to run $", binName, "exec stop")
	if lib.HadExecError {
		lib.ColorERR.Println(err)
		return statusFailedExec
	}
	return retryOrReport()
}

func retryOrReport() int {
	const issues = "https://github.com/" + githubSlug + "/issues"
	const email = "ook@fuzzymonkey.co"
	w := os.Stderr
	fmt.Fprintln(w, "\nLooks like something went wrong... Maybe try again with -v?")
	fmt.Fprintf(w, "\nYou may want to take a look at %s\n", lib.LogID())
	fmt.Fprintf(w, "or come by %s\n", issues)
	fmt.Fprintf(w, "or drop us a line at %s\n", email)
	fmt.Fprintln(w, "\nThank you for your patience & sorry about this :)")
	return statusFailed
}
