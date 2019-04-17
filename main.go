package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/lib"
	"github.com/docopt/docopt-go"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/mapstructure"
)

//go:generate echo Let's go bananas!
//go:generate ./misc/gen_meta.sh

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
	binTitle   = binName + "/" + binVersion
	githubSlug = "FuzzyMonkeyCo/" + binName

	// Environment variables used
	envAPIKey = "FUZZYMONKEY_API_KEY"
)

var (
	clientUtils = &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)
}

func main() {
	os.Exit(actualMain())
}

type params struct {
	Fuzz, Shrink             bool
	Lint, Schema             bool
	Init, Env, Login         bool
	Exec, Start, Reset, Stop bool
	Update                   bool     `mapstructure:"--update"`
	HideConfig               bool     `mapstructure:"--hide-config"`
	ShowSpec                 bool     `mapstructure:"--show-spec"`
	N                        uint32   `mapstructure:"--tests"`
	Verbosity                uint8    `mapstructure:"-v"`
	ValidateAgainst          string   `mapstructure:"--validate-against"`
	EnvVars                  []string `mapstructure:"VAR"`
	FuzzOnly                 []string `mapstructure:"--only"`
	FuzzExcept               []string `mapstructure:"--except"`
}

func usage() (args *params, ret int) {
	B, V, D := lib.ColorNFO.Sprintf(binName), binVersion, binDescribe
	usage := B + "\tv" + V + "\t" + D + "\t" + runtime.Version() + `

Usage:
  ` + B + ` [-vvv] init [--with-magic]
  ` + B + ` [-vvv] env [VAR ...]
  ` + B + ` [-vvv] login --user=USER
  ` + B + ` [-vvv] fuzz [--tests=N] [--seed=SEED] [--tag=TAG]... [--only=REGEX]... [--except=REGEX]...
  ` + B + ` [-vvv] shrink --test=ID [--seed=SEED] [--tag=TAG]...
  ` + B + ` [-vvv] lint [--show-spec] [--hide-config]
  ` + B + ` [-vvv] schema [--validate-against=PTR]
  ` + B + ` [-vvv] exec (start | reset | stop)
  ` + B + ` [-vvv] -h | --help
  ` + B + ` [-vvv]      --update
  ` + B + ` [-vvv] -V | --version

Options:
  -v, -vv, -vvv           Debug verbosity level
  -h, --help              Show this screen
  -U, --update            Ensures ` + B + ` is current
  -V, --version           Show version
  --hide-config           Do not show YAML configuration while linting
  --seed=SEED             Use specific parameters for the RNG
  --validate-against=PTR  Schema $ref to validate STDIN against
  --tag=TAG               Labels that can help classification
  --test=ID               Which test to shrink
  --tests=N               Number of tests to run [default: 100]
  --only=REGEX            Only test matching endpoints
  --except=REGEX          Do not test these endpoints
  --user=USER             Authenticate on fuzzymonkey.co as USER
  --with-magic            Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  ` + B + ` --update
  ` + B + ` init --with-magic
  ` + B + ` fuzz --only=/pets
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
	args, ret := usage()
	if args == nil {
		return ret
	}

	if err := lib.MakePwdID(binName); err != nil {
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

	if args.Update {
		return doUpdate()
	}

	if args.Env {
		return doEnv(args.EnvVars)
	}

	cfg, err := lib.NewCfg(args.Lint && !args.HideConfig)
	if err != nil {
		return retryOrReport()
	}
	if args.Lint {
		e := fmt.Sprintf("%s is a valid v%d configuration", lib.LocalCfg, cfg.Version)
		log.Println("[NFO]", e)
		lib.ColorNFO.Println(e)
	}

	if args.Exec {
		switch {
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
	eids, err := vald.FilterEndpoints(args.FuzzOnly, args.FuzzExcept)
	if err != nil {
		lib.ColorERR.Println(err)
		return statusFailed
	}
	cfg.EIDs = eids
	cfg.N = args.N
	mnk := lib.NewMonkey(cfg, vald, binTitle)
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
	if len(vars) == 0 {
		for key := range all {
			fmt.Printf("%s=\"%s\"\n", key, os.Getenv(key))
		}
		return statusOK
	}

	for _, key := range vars {
		if printed, ok := all[key]; !ok || printed {
			return statusFailed
		}
		all[key] = true
		fmt.Printf("%s=\"%s\"\n", key, os.Getenv(key))
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

	if act := lib.ExecuteScript(cfg, kind); act.Failure || !act.Success {
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
