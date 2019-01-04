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
	binName    = "monkey"
	binTitle   = binName + "/" + binVersion
	envAPIKey  = "FUZZYMONKEY_API_KEY"
	githubSlug = "FuzzyMonkeyCo/" + binName
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
	B, V, D := lib.ColorNFO.Sprintf(binName), binVersion, binDescribe
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
		lib.ColorERR.Println(err)
		ret = 1
		return
	}
	if opts["--version"].(bool) {
		fmt.Println(binTitle)
		return // ret = 0
	}

	args = &params{}
	if err := mapstructure.WeakDecode(opts, args); err != nil {
		lib.ColorERR.Println(err)
		return nil, 1
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
	log.Printf("[ERR] (not an error) %s %s %#v\n", binTitle, lib.LogID(), args)

	if args.Update {
		return doUpdate()
	}

	cfg, err := lib.NewCfg(args.Lint && !args.HideConfig)
	if err != nil {
		return retryOrReport()
	}
	if args.Lint {
		e := fmt.Sprintf("%s is a valid v%d configuration", lib.LocalCfg, cfg.Version)
		log.Println("[NFO]", e)
		lib.ColorNFO.Printf(e)
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
		return 2
	}
	if args.Lint {
		err := fmt.Errorf("%s is a valid %v specification", docPath, cfg.Kind)
		log.Println("[NFO]", err)
		lib.ColorNFO.Println(err)
		return 0
	}

	if args.Schema {
		return doSchema(vald, args.ValidateAgainst)
	}

	if cfg.ApiKey = os.Getenv(envAPIKey); cfg.ApiKey == "" {
		err = fmt.Errorf("$%s is unset", envAPIKey)
		log.Println("[ERR]", err)
		lib.ColorERR.Println(err)
		return retryOrReport()
	}

	cfg.N = args.N
	return doFuzz(cfg, vald)
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

	// assumes not v-prefixed
	// assumes never re-tagging releases
	// assumes only releasing newer tags
	if latest != binVersion {
		fmt.Println("A version newer than", binVersion, "is out:", latest)
		if err := rel.ReplaceCurrentRelease(latest); err != nil {
			fmt.Println("The update failed 🙈 please try again")
			return 3
		}
	}
	return 0
}

func doSchema(vald *lib.Validator, ref string) int {
	refs := vald.Refs
	refsCount := len(refs)
	showRefs := func() {
		for absRef := range refs {
			fmt.Println(absRef)
		}
	}
	if ref == "" {
		log.Printf("[NFO] found %d refs\n", refsCount)
		lib.ColorNFO.Printf("Found %d refs\n", refsCount)
		showRefs()
		return 0
	}

	if err := vald.ValidateAgainstSchema(ref); err != nil {
		switch err {
		case lib.ErrInvalidPayload:
		case lib.ErrNoSuchRef:
			lib.ColorERR.Printf("No such $ref '%s'\n", ref)
			if refsCount > 0 {
				fmt.Println("Try one of:")
				showRefs()
			}
		default:
			lib.ColorERR.Println(err)
		}
		return 9
	}
	lib.ColorNFO.Println("Payload is valid")
	return 0
}

func doExec(cfg *lib.UserCfg, kind lib.ExecKind) int {
	if _, err := os.Stat(lib.Shell()); os.IsNotExist(err) {
		log.Println(lib.Shell(), "is required")
		return 5
	}
	if err := lib.SnapEnv(lib.EnvID()); err != nil {
		return retryOrReport()
	}

	if act := lib.ExecuteScript(cfg, kind); act.Failure || !act.Success {
		return 7
	}
	return 0
}

func doFuzz(cfg *lib.UserCfg, vald *lib.Validator) int {
	if _, err := os.Stat(lib.Shell()); os.IsNotExist(err) {
		log.Printf("[ERR] %s is required\n", lib.Shell())
		return 5
	}

	if err := lib.SnapEnv(lib.EnvID()); err != nil {
		return retryOrReport()
	}

	if err := lib.NewWS(cfg, wsURL, binTitle); err != nil {
		return retryOrReport()
	}
	act := lib.Action(&lib.DoFuzz{})
	mnk := lib.NewMonkey(cfg, vald, binTitle)

	for {
		if done, ok := act.(*lib.FuzzProgress); ok && (done.GetFailure() || done.GetSuccess()) {
			ensureDeleted(lib.EnvID())
			return done.Outcome(mnk)
		}

		var err error
		if act, err = lib.FuzzNext(mnk, act); err != nil {
			return retryOrReportThenCleanup(err)
		}
	}
}

func retryOrReportThenCleanup(err error) int {
	defer func() {
		lib.ColorWRN.Println("You might want to run $ monkey exec stop")
	}()
	if lib.HadExecError {
		return 7
	}
	return retryOrReport()
}

func retryOrReport() int {
	const issues = "https://github.com/" + githubSlug + "/issues"
	const email = "ook@fuzzymonkey.co"
	fmt.Println("\nLooks like something went wrong... Maybe try again with -v?")
	fmt.Printf("\nYou may want to take a look at %s\n", lib.LogID())
	fmt.Printf("or come by %s\n", issues)
	fmt.Printf("or drop us a line at %s\n", email)
	fmt.Println("\nThank you for your patience & sorry about this :)")
	return 1
}
