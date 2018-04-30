package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/docopt/docopt-go"
	"github.com/hashicorp/logutils"
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
	apiRoot     string
	initURL     string
	nextURL     string
	lintURL     string
	clientUtils = &http.Client{}
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	if binVersion == "0.0.0" {
		apiRoot = "http://fuzz.dev.fuzzymonkey.co/1"
		lintURL = "http://lint.dev.fuzzymonkey.co/1/blob"
	} else {
		//FIXME: use HTTPS
		apiRoot = "http://fuzz.fuzzymonkey.co/1"
		lintURL = "http://lint.fuzzymonkey.co/1/blob"
	}
	initURL = apiRoot + "/init"
	nextURL = apiRoot + "/next"

	loadSchemas()
}

func main() {
	os.Exit(actualMain())
}

func usage() (docopt.Opts, error) {
	usage := binName + "\tv" + binVersion + "\t" + binDescribe + "\t" + runtime.Version() + `

Usage:
  ` + binName + ` [-vvv] fuzz
  ` + binName + ` [-vvv] lint
  ` + binName + ` [-vvv] exec (start | reset | stop)
  ` + binName + ` [-vvv] -h | --help
  ` + binName + ` [-vvv] -U | --update
  ` + binName + ` [-vvv] -V | --version

Options:
  -v, -vv, -vvv  Debug verbosity level
  -h, --help     Show this screen
  -U, --update   Ensures ` + binName + ` is latest
  -V, --version  Show version

Try:
     export FUZZYMONKEY_API_KEY=42
  ` + binName + ` --update
  ` + binName + ` fuzz`

	parser := &docopt.Parser{
		HelpHandler:  docopt.PrintHelpOnly,
		OptionsFirst: true,
	}
	return parser.ParseArgs(usage, os.Args[1:], binTitle)
}

func actualMain() int {
	args, err := usage()
	if err != nil {
		// Usage shown: bad args
		return 1
	}
	if len(args) == 0 {
		// Help or version shown
		return 0
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
		MinLevel: logLevel(args["-v"].(int)),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Println("[ERR] (not an error)", binTitle, logID(), args)

	if args["--update"].(bool) {
		return doUpdate()
	}

	cfg, err := newCfg()
	if err != nil || cfg == nil {
		return retryOrReport()
	}

	if args["exec"].(bool) {
		switch {
		case args["start"].(bool):
			return doExec(cfg, kindStart)
		case args["reset"].(bool):
			return doExec(cfg, kindReset)
		}
		return doExec(cfg, kindStop)
	}

	apiKey := os.Getenv(envAPIKey)
	// Always lint before fuzzing
	validSpec, err := lintDocs(cfg, apiKey)
	if err != nil {
		return 2
	}
	if args["lint"].(bool) {
		return 0
	}

	return doFuzz(cfg, apiKey, validSpec)
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		fmt.Println(err)
		log.Panic("[ERR] ", err)
	}
}

func logLevel(verbosity int) logutils.LogLevel {
	lvl := map[int]string{
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

func doFuzz(cfg *ymlCfg, apiKey string, spec []byte) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Printf("%s is required\n", shell())
		return 5
	}

	if err := snapEnv(envID()); err != nil {
		return retryOrReport()
	}

	if apiKey == "" {
		log.Printf("$%s is unset\n", envAPIKey)
		return 4
	}

	cmd, err := newFuzz(cfg, apiKey, spec)
	if err != nil {
		return retryOrReportThenCleanup(cfg, err)
	}

	for {
		if cmd.Kind() == kindDone {
			maybePostStop(cfg)
			ensureDeleted(envID())
			return fuzzOutcome(cmd.(*doneCmd))
		}

		if cmd, err = next(cfg, cmd); err != nil {
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
	issues := "https://github.com/" + githubSlug + "/issues"
	email := "ook@fuzzymonkey.co"
	fmt.Println("\nLooks like something went wrong... Maybe try again with -v?")
	fmt.Printf("\nYou may want to take a look at %s\n", logID())
	fmt.Printf("or come by %s\n", issues)
	fmt.Printf("or drop us a line at %s\n", email)
	fmt.Println("\nThank you for your patience & sorry about this :)")
	return 1
}
