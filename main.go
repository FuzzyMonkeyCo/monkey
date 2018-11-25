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
	colorNFO = color.New(color.Bold)
}

func main() {
	os.Exit(actualMain())
}

type params struct {
	Fuzz, Shrink             bool
	Lint, Schema             bool
	Init, Login              bool
	Exec, Start, Reset, Stop bool
	Update                   bool   `mapstructure:"--update"`
	HideConfig               bool   `mapstructure:"--hide-config"`
	ShowSpec                 bool   `mapstructure:"--show-spec"`
	N                        uint32 `mapstructure:"--tests"`
	Verbosity                uint8  `mapstructure:"-v"`
	ValidateAgainst          string `mapstructure:"--validate-against"`
}

func usage() (args *params, ret int) {
	B, V, D := colorNFO.Sprintf(binName), binVersion, binDescribe
	usage := B + "\tv" + V + "\t" + D + "\t" + runtime.Version() + `

Usage:
  ` + B + ` [-vvv] init [--with-magic]
  ` + B + ` [-vvv] login --user=USER
  ` + B + ` [-vvv] fuzz [--tests=N] [--seed=SEED] [--tag=TAG]...
  ` + B + ` [-vvv] shrink --test=ID [--seed=SEED] [--tag=TAG]...
  ` + B + ` [-vvv] lint [--show-spec] [--hide-config]
  ` + B + ` [-vvv] schema [--validate-against=REF]
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
  --validate-against=REF  Schema $ref to validate STDIN against
  --tag=TAG               Labels that can help classification
  --test=ID               Which test to shrink
  --tests=N               Number of tests to run [default: 100]
  --user=USER             Authenticate on fuzzymonkey.co as USER
  --with-magic            Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  ` + B + ` --update
  ` + B + ` init --with-magic
  ` + B + ` fuzz
  echo '"kitty"' | ` + B + ` schema --validate-against '#/components/schemas/PetKind'`

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

	config, err := readCfg()
	if err != nil {
		return retryOrReport()
	}
	cfg, err := newCfg(config, args.Lint && !args.HideConfig)
	if err != nil {
		return retryOrReport()
	}
	if args.Lint {
		e := fmt.Sprintf("%s is a valid v%d configuration", localCfg, cfg.Version)
		log.Println("[NFO]", e)
		colorNFO.Printf(e)
	}

	if args.Exec {
		switch {
		case args.Start:
			return doExec(cfg, ExecKind_start)
		case args.Reset:
			return doExec(cfg, ExecKind_reset)
		case args.Stop:
			return doExec(cfg, ExecKind_stop)
		default:
			return retryOrReport()
		}
	}

	docPath, blob, err := cfg.findThenReadBlob()
	if err != nil {
		return retryOrReport()
	}

	// Always lint before fuzzing
	vald, err := doLint(docPath, blob, args.ShowSpec)
	if err != nil {
		return 2
	}
	if args.Lint {
		err := fmt.Errorf("%s is a valid %v specification", docPath, cfg.Kind)
		log.Println("[NFO]", err)
		colorNFO.Println(err)
		return 0
	}

	if args.Schema {
		return doSchema(vald, args.ValidateAgainst)
	}

	if cfg.ApiKey = os.Getenv(envAPIKey); cfg.ApiKey == "" {
		err = fmt.Errorf("$%s is unset", envAPIKey)
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return retryOrReport()
	}

	cfg.N = args.N
	return doFuzz(cfg, vald)
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		log.Println("[ERR]", err)
		colorERR.Println(err)
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
	latest, err := peekLatestRelease()
	if err != nil {
		return retryOrReport()
	}

	// assumes not v-prefixed
	// assumes never re-tagging releases
	if latest != binVersion {
		fmt.Println("A version newer than", binVersion, "is out:", latest)
		if err := replaceCurrentRelease(latest); err != nil {
			fmt.Println("The update failed 🙈 please try again")
			return 3
		}
	}
	return 0
}

func doSchema(vald *validator, ref string) int {
	refs := vald.Refs
	refsCount := len(refs)
	showRefs := func() {
		for absRef := range refs {
			fmt.Println(absRef)
		}
	}
	if ref == "" {
		log.Printf("[NFO] found %d refs\n", refsCount)
		colorNFO.Printf("Found %d refs\n", refsCount)
		showRefs()
		return 0
	}

	if err := vald.validateAgainstSchema(ref); err != nil {
		switch err {
		case errInvalidPayload:
		case errNoSuchRef:
			colorERR.Printf("No such $ref '%s'\n", ref)
			if refsCount > 0 {
				fmt.Println("Try one of:")
				showRefs()
			}
		default:
			colorERR.Println(err)
		}
		return 9
	}
	colorNFO.Println("Payload is valid")
	return 0
}

func doExec(cfg *UserCfg, kind ExecKind) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Println(shell(), "is required")
		return 5
	}
	if err := snapEnv(envID()); err != nil {
		return retryOrReport()
	}

	if act := executeScript(cfg, kind); act.Failure || !act.Success {
		return 7
	}
	return 0
}

func doFuzz(cfg *UserCfg, vald *validator) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Printf("[ERR] %s is required\n", shell())
		return 5
	}

	if err := snapEnv(envID()); err != nil {
		return retryOrReport()
	}

	if err := newWS(cfg); err != nil {
		return retryOrReport()
	}
	act := action(&DoFuzz{})
	mnk := &monkey{cfg: cfg, vald: vald}

	for {
		if done, ok := act.(*FuzzProgress); ok && (done.GetFailure() || done.GetSuccess()) {
			ensureDeleted(envID())
			return fuzzOutcome(done)
		}

		var err error
		if act, err = fuzzNext(mnk, act); err != nil {
			return retryOrReportThenCleanup(err)
		}
	}
}

func retryOrReportThenCleanup(err error) int {
	defer func() { colorWRN.Println("You might want to run $ monkey exec stop") }()
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
