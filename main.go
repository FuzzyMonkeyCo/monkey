package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/fatih/color"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/mapstructure"
)

//go:generate echo Let's go bananas!
//go:generate go run misc/gen_schemas.go
//go:generate ./misc/gen_meta.sh

const (
	binName    = "monkey"
	binTitle   = binName + "/" + binVersion
	envAPIKey  = "FUZZYMONKEY_API_KEY"
	githubSlug = "FuzzyMonkeyCo/" + binName
)

var (
	clientUtils = &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	colorERR *color.Color
	colorWRN *color.Color
	colorNFO *color.Color
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	colorERR = color.New(color.FgRed)
	colorWRN = color.New(color.FgYellow)
	colorNFO = color.New(color.FgWhite, color.Bold)

	loadSchemas()
}

func main() {
	os.Exit(actualMain())
}

type params struct {
	Init, Login, Fuzz, Shrink, Lint bool
	Exec, Start, Reset, Stop        bool
	Update                          bool `mapstructure:"--update"`
	HideConfig                      bool `mapstructure:"--hide-config"`
	ShowSpec                        bool `mapstructure:"--show-spec"`
	N                               uint `mapstructure:"--tests"`
	Verbosity                       uint `mapstructure:"-v"`
}

func usage() (args *params, ret int) {
	usage := binName + "\tv" + binVersion + "\t" + binDescribe + "\t" + runtime.Version() + `

Usage:
  ` + binName + ` [-vvv] init [--with-magic]
  ` + binName + ` [-vvv] login --user=USER
  ` + binName + ` [-vvv] fuzz [--tests=N] [--seed=SEED] [--tag=TAG]...
  ` + binName + ` [-vvv] shrink --test=ID [--seed=SEED] [--tag=TAG]...
  ` + binName + ` [-vvv] lint [--show-spec] [--hide-config]
  ` + binName + ` [-vvv] exec (start | reset | stop)
  ` + binName + ` [-vvv] -h | --help
  ` + binName + ` [-vvv]      --update
  ` + binName + ` [-vvv] -V | --version

Options:
  -v, -vv, -vvv  Debug verbosity level
  -h, --help     Show this screen
  -U, --update   Ensures ` + binName + ` is latest
  -V, --version  Show version
  --hide-config  Do not show YAML configuration while linting
  --seed=SEED    Use specific parameters for the RNG
  --tag=TAG      Labels that can help classification
  --test=ID      Which test to shrink
  --tests=N      Number of tests to run [default: 100]
  --user=USER    Authenticate on fuzzymonkey.co as USER
  --with-magic   Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  ` + binName + ` --update
  ` + binName + ` init --with-magic
  ` + binName + ` fuzz`

	// https://github.com/docopt/docopt.go/issues/59
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		// Usage shown: bad args
		colorERR.Println(err)
		ret = 1
		return
	}
	if opts["--version"].(bool) {
		fmt.Println(binTitle)
		return // ret = 0
	}

	args = &params{}
	if err := mapstructure.WeakDecode(opts, args); err != nil {
		colorERR.Println(err)
		return nil, 1
	}
	return
}

func actualMain() int {
	args, ret := usage()
	if args == nil {
		return ret
	}

	if err := makePwdID(); err != nil {
		return retryOrReport()
	}

	logCatchall, err := os.OpenFile(logID(), os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Println(err)
		return retryOrReport()
	}
	defer logCatchall.Close()
	logFiltered := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DBG", "NFO", "ERR", "NOP"},
		MinLevel: logLevel(args.Verbosity),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Printf("[ERR] (not an error) %s %s %#v\n", binTitle, logID(), args)

	if args.Update {
		return doUpdate()
	}

	yml, err := readYML()
	if err != nil {
		return retryOrReport()
	}
	cfg, err := newCfg(yml, args.Lint && !args.HideConfig)
	if err != nil || cfg == nil {
		return retryOrReport()
	}

	if args.Exec {
		switch {
		case args.Start:
			return doExec(cfg, kindStart)
		case args.Reset:
			return doExec(cfg, kindReset)
		}
		return doExec(cfg, kindStop)
	}

	docPath, blob, err := cfg.findThenReadBlob()
	if err != nil {
		return retryOrReport()
	}

	// Always lint before fuzzing
	validSpec, err := doLint(docPath, blob, args.ShowSpec)
	if err != nil {
		return 2
	}
	log.Println("[NFO] No validation errors found.")
	colorNFO.Println("No validation errors found.")
	if args.Lint {
		return 0
	}

	apiKey := os.Getenv(envAPIKey)
	if err := doAuth(cfg, apiKey, args.N); err != nil {
		return retryOrReport()
	}

	return doFuzz(cfg, validSpec)
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		log.Println("[ERR]", err)
		colorERR.Println(err)
		panic(err)
	}
}

func logLevel(verbosity uint) logutils.LogLevel {
	lvl := map[uint]string{
		0: "NOP",
		1: "ERR",
		2: "NFO",
		3: "DBG",
	}[verbosity]
	return logutils.LogLevel(lvl)
}

func doUpdate() int {
	latest, err := peekLatestRelease()
	if err != nil {
		return retryOrReport()
	}

	// assumes not v-prefixed
	// assumes never patching old-minor releases
	if latest != binVersion {
		fmt.Printf("A newer version of %s is out: %s (you have %s)\n",
			binName, latest, binVersion)
		if err := replaceCurrentRelease(latest); err != nil {
			fmt.Println("The update failed 🙈 please try again")
			return 3
		}
	}
	return 0
}

func doExec(cfg *ymlCfg, kind cmdKind) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Printf("%s is required\n", shell())
		return 5
	}
	if err := snapEnv(envID()); err != nil {
		return retryOrReport()
	}

	if cmdRep := executeScript(cfg, kind); cmdRep.Failed {
		return 7
	}
	return 0
}

func doFuzz(cfg *ymlCfg, spec *SpecIR) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Printf("%s is required\n", shell())
		return 5
	}

	if err := snapEnv(envID()); err != nil {
		return retryOrReport()
	}

	cmd, err := newFuzz(cfg, spec)
	if err != nil {
		return retryOrReportThenCleanup(cfg, err)
	}

	for {
		if cmd.Kind() == kindDone {
			maybePostStop(cfg)
			ensureDeleted(envID())
			return fuzzOutcome(cmd.(*doneCmd))
		}

		if cmd, err = fuzzNext(cfg, cmd); err != nil {
			return retryOrReportThenCleanup(cfg, err)
		}
	}
}

func retryOrReportThenCleanup(cfg *ymlCfg, err error) int {
	defer maybePostStop(cfg)
	if hadExecError {
		return 7
	}
	return retryOrReport()
}

func retryOrReport() int {
	const issues = "https://github.com/" + githubSlug + "/issues"
	const email = "ook@fuzzymonkey.co"
	fmt.Println("\nLooks like something went wrong... Maybe try again with -v?")
	fmt.Printf("\nYou may want to take a look at %s\n", logID())
	fmt.Printf("or come by %s\n", issues)
	fmt.Printf("or drop us a line at %s\n", email)
	fmt.Println("\nThank you for your patience & sorry about this :)")
	return 1
}
