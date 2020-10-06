package main

import (
	"fmt"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/code"
	docopt "github.com/docopt/docopt-go"
	"github.com/mitchellh/mapstructure"
)

type params struct {
	Fuzz, Lint, Schema             bool
	Init, Env, Login, Logs         bool
	Update, Version                bool
	Exec, Start, Reset, Stop, Repl bool
	ShowSpec                       bool          `mapstructure:"--show-spec"`
	Seed                           string        `mapstructure:"--seed"`
	Shrink                         string        `mapstructure:"--shrink"`
	Tags                           []string      `mapstructure:"--tag"`
	N                              uint32        `mapstructure:"--intensity"`
	Verbosity                      uint8         `mapstructure:"-v"`
	LogOffset                      uint64        `mapstructure:"--previous"`
	ValidateAgainst                string        `mapstructure:"--validate-against"`
	EnvVars                        []string      `mapstructure:"VAR"`
	OverallBudgetTime              time.Duration `mapstructure:"--time-budget-overall"`
}

func usage() (args *params, ret int) {
	B := as.ColorNFO.Sprintf(binName)
	usage := binTitle + `

Usage:
  ` + B + ` [-vvv] init [--with-magic]
  ` + B + ` [-vvv] login [--user=USER]
  ` + B + ` [-vvv] fuzz [--intensity=N] [--shrink=ID] [--seed=SEED] [--tag=KV]...
                     [--time-budget-overall=DURATION]
                     [--only=REGEX]... [--except=REGEX]...
                     [--calls-with-input=SCHEMA]... [--calls-without-input=SCHEMA]...
                     [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  ` + B + ` [-vvv] lint [--show-spec]
  ` + B + ` [-vvv] schema [--validate-against=REF]
  ` + B + ` [-vvv] exec (repl | start | reset | stop)
  ` + B + ` [-vvv] env [VAR ...]
  ` + B + ` logs [--previous=N]
  ` + B + ` [-vvv] update
  ` + B + ` version | --version
  ` + B + ` help    | --help    | -h

Options:
  -v, -vv, -vvv                   Debug verbosity level
  version                         Show the version string
  update                          Ensures ` + B + ` is the latest version
  --intensity=N                   The higher the more complex the tests [default: 10]
  --time-budget-overall=DURATION  Stop testing after DURATION (e.g. '30s' or '5h')
  --seed=SEED                     Use specific parameters for the Random Number Generator
  --shrink=ID                     Which failed test to minimize
  --tag=KV                        Labels that can help classification (format: key=value)
  --only=REGEX                    Only test matching calls
  --except=REGEX                  Do not test these calls
  --calls-with-input=SCHEMA       Test calls which can take schema PTR as input
  --calls-without-output=SCHEMA   Test calls which never output schema PTR
  --user=USER                     Authenticate on fuzzymonkey.co as USER
  --validate-against=REF          Schema $ref to validate STDIN against
  --with-magic                    Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  ` + B + ` update
  ` + B + ` exec reset
  ` + B + ` fuzz --only /pets --calls-without-input=NewPet
  echo '"kitty"' | ` + B + ` schema --validate-against=#/components/schemas/PetKind`

	// https://github.com/docopt/docopt.go/issues/59
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		// Usage shown: bad args
		as.ColorERR.Println(err)
		ret = code.Failed
		return
	}

	if opts["help"].(bool) {
		fmt.Println(usage)
		ret = code.OK
		return
	}

	args = &params{}
	cfg := &mapstructure.DecoderConfig{
		Result:           &args,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	}
	d, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		as.ColorERR.Println(err)
		return nil, code.Failed
	}
	if err := d.Decode(opts); err != nil {
		as.ColorERR.Println(err)
		return nil, code.Failed
	}

	if opts["--version"].(bool) {
		args.Version = true
	}
	if args.Fuzz && args.N == 0 {
		// TODO: upstream docopt doesn't handle [default: 10]
		args.N = 10
	}

	return
}
