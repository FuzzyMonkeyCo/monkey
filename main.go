package main

import (
	"github.com/docopt/docopt-go"
	"log"
)

//go:generate go run misc/include_jsons.go

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)
}

const (
	SemVer  = "0.1.0"
	Version = "manlion/" + SemVer
)

func usage() (map[string]interface{}, error) {
	usage := `manlion

Usage:
  manlion test [--slow]
  manlion -h | --help
  manlion --version

Options:
  --slow        Don't phone home using Websockets
  -h --help     Show this screen
  --version     Show version`

	return docopt.Parse(usage, nil, false, Version, false)
}

func main() {
	args, err := usage()
	if err != nil {
		log.Fatal("!args: ", err)
	}
	//FIXME: use args
	log.Println(args)

	// latest := GetLatestRelease()
	// log.Printf("%s >? %s: %v\n", latest, SemVer, IsOutOfDate(SemVer,latest))

	cfg, cmdData := initDialogue()
	log.Println("cmdData:", string(cmdData))

	var done bool
	for {
		cmdData, done = next(cfg, cmdData)
		if done {
			break
		}
	}

}
