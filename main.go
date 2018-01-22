package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
	docsURL     string
	clientUtils = &http.Client{}
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	if binVersion == "0.0.0" {
		apiRoot = "http://fuzz.dev.fuzzymonkey.co/1"
		docsURL = "http://lint.dev.fuzzymonkey.co/1/blob"
	} else {
		//FIXME: use HTTPS
		apiRoot = "http://fuzz.fuzzymonkey.co/1"
		docsURL = "http://lint.fuzzymonkey.co/1/blob"
	}
	initURL = apiRoot + "/init"
	nextURL = apiRoot + "/next"

	loadSchemas()

	makePwdID()
}

func main() {
	os.Exit(actualMain())
}

func usage() (docopt.Opts, error) {
	usage := binName + " v" + binVersion + " " + binDescribe + `

Usage:
  ` + binName + ` [-vvv] fuzz
  ` + binName + ` [-vvv] validate
  ` + binName + ` -h | --help
  ` + binName + ` -U | --update
  ` + binName + ` -V | --version

Options:
  -v, -vv, -vvv  Verbosity level
  -h, --help     Show this screen
  -U, --update   Ensures ` + binName + ` is latest
  -V, --version  Show version

Try:
                         ` + binName + ` --update
  FUZZYMONKEY_API_KEY=42 ` + binName + ` -v fuzz`

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

	logFile := pwdID + ".log"
	logCatchall, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE, 0640)
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
	log.Println("[ERR]", binTitle, logFile, args)

	if args["--update"].(bool) {
		return doUpdate()
	}

	apiKey := os.Getenv(envAPIKey)
	if args["validate"].(bool) {
		return doValidate(apiKey)
	}

	// if args["fuzz"].(bool)
	return doFuzz(apiKey)
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		fmt.Println(err)
		log.Panic("[ERR] ", err)
	}
}

func logLevel(verbosity int) logutils.LogLevel {
	var lvl string
	switch verbosity {
	case 1:
		lvl = "ERR"
	case 2:
		lvl = "NFO"
	case 3:
		lvl = "DBG"
	default:
		lvl = "NOP"
	}
	return logutils.LogLevel(lvl)
}

func doUpdate() int {
	latest, err := getLatestRelease()
	if err != nil {
		return retryOrReport()
	}

	// assumes not v-prefixed
	// assumes new releases are only for newer software
	if latest != binVersion {
		fmt.Printf("A newer version of %s is out: %s (you have %s)\n", binName, latest, binVersion)
		if err := replaceCurrentRelease(latest); err != nil {
			return 3
		}
	}
	return 0
}

func doValidate(apiKey string) int {
	if yml, err := readYML(); err == nil {
		if _, err := validateDocs(apiKey, yml); err != nil {
			return 2
		}
		return 0
	}
	return retryOrReport()
}

func doFuzz(apiKey string) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Printf("%s is required\n", shell())
		return 5
	}

	if apiKey == "" {
		log.Printf("$%s is unset\n", envAPIKey)
		return 4
	}

	envSerializedPath := pwdID + ".env"
	if err := snapEnv(envSerializedPath); err != nil {
		return retryOrReport()
	}

	cfg, cmd, err := initDialogue(apiKey)
	if err != nil {
		if _, ok := err.(*docsInvalidError); ok {
			ensureDeleted(envSerializedPath)
			return 2
		}
		if cfg == nil {
			return retryOrReport()
		}
		return retryOrReportThenCleanup(cfg)
	}

	for {
		if cmd.Kind() == "done" {
			ensureDeleted(envSerializedPath)
			return fuzzOutcome(cmd.(*doneCmd))
		}

		if cmd, err = next(cfg, cmd); err != nil {
			return retryOrReportThenCleanup(cfg)
		}
	}
}

func retryOrReportThenCleanup(cfg *ymlCfg) int {
	exitCode := retryOrReport()
	maybePostStop(cfg)
	return exitCode
}

func retryOrReport() int {
	issues := "https://github.com/" + githubSlug + "/issues"
	email := "ook@fuzzymonkey.co"
	fmt.Println("\nLooks like something went wrong... Maybe try again with -v?")
	fmt.Printf("\nYou may want to take a look at %s.log\n", pwdID)
	fmt.Printf("or come by %s\n", issues)
	fmt.Printf("or drop us a line at %s\n", email)
	fmt.Println("\nThank you for your patience & sorry about this :)")
	return 1
}
