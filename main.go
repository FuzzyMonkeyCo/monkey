package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"gopkg.in/docopt/docopt.go.v0"
	"gopkg.in/hashicorp/logutils.v0"
)

//go:generate go run misc/include_jsons.go

const (
	binName   = "testman"
	binTitle  = binName + "/" + binVersion
	envAPIKey = "COVEREDCI_API_KEY"
)

var (
	isDebug     bool
	apiRoot     string
	initURL     string
	nextURL     string
	docsURL     string
	clientUtils = &http.Client{}
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	isDebug = "0.0.0" == binVersion

	if isDebug {
		apiRoot = "http://localhost:1042/1"
		docsURL = "http://localhost:2042/1/blob"
	} else {
		apiRoot = "https://test.coveredci.com/1"
		docsURL = "https://lint.coveredci.com/1/blob"
	}
	initURL = apiRoot + "/init"
	nextURL = apiRoot + "/next"

	unstacheInit()

	loadSchemas()

	makePwdID()
}

func main() {
	os.Exit(actualMain())
}

func usage() (map[string]interface{}, error) {
	usage := `testman

Usage:
  testman [-vvv] test
  testman [-vvv] validate
  testman -h | --help
  testman -V | --version

Options:
  -v, -vv, -vvv  Verbosity level
  -h, --help     Show this screen
  -V, --version  Show version`

	return docopt.Parse(usage, nil, true, binTitle, true)
}

func actualMain() int {
	args, err := usage()
	if err != nil {
		log.Println("[ERR] !args: ", err)
		return retryOrReport()
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
		MinLevel: logLevel(args),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Println("[ERR]", binTitle, logFile, args)

	if !isDebug {
		if code := isRunningLatest(); code != 0 {
			return code
		}
	}

	apiKey := getAPIKey()
	if args["validate"].(bool) {
		if yml, err := readYML(); err == nil {
			if _, err := validateDocs(apiKey, yml); err != nil {
				return 2
			}
			return 0
		}
		return retryOrReport()
	}

	// args["test"].(bool) = true
	return doTest(apiKey)
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		fmt.Println(err)
		log.Panic("[ERR] ", err)
	}
}

func getAPIKey() string {
	apiKey := os.Getenv(envAPIKey)
	if isDebug {
		apiKey = "42"
	}
	return apiKey
}

func logLevel(args map[string]interface{}) logutils.LogLevel {
	var lvl string
	switch args["-v"].(int) {
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

func isRunningLatest() int {
	latest, err := getLatestRelease()
	if err != nil {
		return retryOrReport()
	}

	ko, err := isOutOfDate(binVersion, latest)
	if err != nil {
		return retryOrReport()
	}
	if ko {
		err := fmt.Errorf("A newer version of %s is out: %s (you have %s)", binName, latest, binVersion)
		log.Println("[ERR]", err)
		fmt.Println(err)
		return 3
	}

	return 0
}

func doTest(apiKey string) int {
	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Println(shell() + " is required")
		return 5
	}

	if apiKey == "" {
		log.Println("$" + envAPIKey + " is unset")
		return 4
	}

	envSerializedPath := pwdID + ".env"
	ensureDeleted(envSerializedPath)
	if err := snapEnv(envSerializedPath); err != nil {
		return retryOrReport()
	}
	defer ensureDeleted(envSerializedPath)

	cfg, cmd, err := initDialogue(apiKey)
	if err != nil {
		if _, ok := err.(*docsInvalidError); ok {
			return 2
		}
		return retryOrReport()
	}

	for {
		if cmd.Kind() == "done" {
			return testOutcome(cmd.(doneCmd))
		}

		if cmd, err = next(cfg, cmd); err != nil {
			return retryOrReport()
		}
	}
}

func retryOrReport() int {
	issues := "https://github.com/CoveredCI/testman/issues"
	email := "hi@coveredci.co"
	fmt.Println("Looks like something went wrong... Maybe try again?")
	fmt.Printf("\tYou may want to take a look at %s.log\n", pwdID)
	fmt.Printf("\tor come by %s\n", issues)
	fmt.Printf("\tor drop us a line at %s\n", email)
	fmt.Println("Thanks & sorry about this :)")
	return 1
}
