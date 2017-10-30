package main

import (
	"log"
	"os"

	"gopkg.in/docopt/docopt.go.v0"
)

//go:generate go run misc/include_jsons.go

const (
	pkgVersion = "0.2.0"
	pkgTitle   = "testman/" + pkgVersion
	isDebug    = false
)

var (
	apiRoot string
	initURL string
	nextURL string
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	if isDebug {
		apiRoot = "http://localhost:1042" //FIXME use HTTPS
	} else {
		apiRoot = "https://testman.coveredci.com"
	}
	initURL = apiRoot + "/1/init"
	nextURL = apiRoot + "/1/next"

	unstacheInit()
}

func usage() (map[string]interface{}, error) {
	usage := `testman

Usage:
  testman test [--slow]
  testman -h | --help
  testman --version

Options:
  --slow        Don't phone home using Websockets
  -h --help     Show this screen
  --version     Show version`

	return docopt.Parse(usage, nil, true, pkgTitle, false)
}

func main() {
	args, err := usage()
	if err != nil {
		log.Fatal("!args: ", err)
	}
	log.Println(args) //FIXME: use args

	if !isDebug {
		latest := getLatestRelease()
		if isOutOfDate(pkgVersion, latest) {
			log.Fatalf("A newer version of %s is available: %s\n", pkgTitle, latest)
		}
	}

	apiKey := os.Getenv("COVEREDCI_API_KEY")
	if apiKey == "" {
		log.Fatal("$COVEREDCI_API_KEY is unset")
	}

	envSerializedPath := uniquePath()
	ensureDeleted(envSerializedPath)
	snapEnv(envSerializedPath)
	defer ensureDeleted(envSerializedPath)

	cfg, cmd := initDialogue(apiKey)
	log.Printf("cmd: %+v\n", cmd)
	for {
		cmd = next(cfg, cmd)
		if nil == cmd {
			log.Println("We're done!")
			break
		}
	}
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
}
