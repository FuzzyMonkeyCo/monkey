package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/code"
	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	rt "github.com/FuzzyMonkeyCo/monkey/pkg/runtime"
	"github.com/FuzzyMonkeyCo/monkey/pkg/update"
	"github.com/hashicorp/logutils"
)

const (
	binName    = "monkey"
	githubSlug = "FuzzyMonkeyCo/" + binName

	// Environment variables used
	envAPIKey = "FUZZYMONKEY_API_KEY"
)

var (
	binSHA     = "feedb065"
	binVersion = "M.m.p"
	binTitle   = strings.Join([]string{binName, binVersion, binSHA,
		runtime.Version(), runtime.GOOS, runtime.GOARCH}, "\t")
)

func main() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)
	os.Exit(actualMain())
}

func actualMain() int {
	start := time.Now()
	args, ret := usage()
	if args == nil {
		return ret
	}

	if args.Version {
		fmt.Println(binTitle)
		return code.OK
	}

	if args.Logs {
		offset := args.LogOffset
		if offset == 0 {
			offset = 1
		}
		return doLogs(offset)
	}

	if err := cwid.MakePwdID(binName, 0); err != nil {
		return retryOrReport()
	}
	logCatchall, err := os.OpenFile(cwid.LogFile(), os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		as.ColorERR.Println(err)
		return retryOrReport()
	}
	defer logCatchall.Close()
	logFiltered := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DBG", "NFO", "ERR", "NOP"},
		MinLevel: logLevel(args.Verbosity),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Printf("[ERR] (not an error) %s %s %#v", binTitle, cwid.LogFile(), args)
	defer func() { log.Printf("[ERR] (not an error) ran for %s", time.Since(start)) }()

	if args.Init || args.Login {
		// FIXME: implement init & login
		as.ColorERR.Println("Action not implemented yet")
		return code.Failed
	}

	if args.Update {
		return doUpdate()
	}

	if args.Env {
		return doEnv(args.EnvVars)
	}

	if args.Fmt {
		if err := rt.Format(args.FmtW); err != nil {
			if e, ok := err.(rt.FmtError); ok {
				for i := 0; i < len(e); i += 3 {
					as.ColorNFO.Printf("%s ", e[i])
					as.ColorOK.Printf("%s ", e[i+1])
					as.ColorERR.Printf("%s\n", e[i+2])
				}
			} else {
				as.ColorERR.Println(err)
			}
			return code.FailedFmt
		}
		return code.OK
	}

	mrt, err := rt.NewMonkey(binTitle, args.Tags, args.Verbosity)
	if err != nil {
		as.ColorERR.Println(err)
		return code.Failed
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if timeout := args.OverallBudgetTime; timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)
	go func() {
		select {
		case <-ctx.Done():
			log.Println("[NFO] background context done")
			signal.Stop(sigC)
		case <-sigC:
			log.Println("[NFO] received ^C: terminating")
			cancel()
		}
	}()

	// Always lint
	if err := mrt.Lint(ctx, args.ShowSpec); err != nil {
		as.ColorERR.Println(err)
		return code.FailedLint
	}
	if args.Lint {
		e := "Configuration is valid."
		log.Println("[NFO]", e)
		as.ColorNFO.Println(e)
		return code.OK
	}

	if args.Exec {
		var fn func() error
		switch {
		case args.Start:
			fn = mrt.JustExecStart
		case args.Reset:
			fn = mrt.JustExecReset
		case args.Stop:
			fn = mrt.JustExecStop
		case args.Repl:
			fn = mrt.JustExecREPL
		}
		if err := fn(); err != nil {
			as.ColorERR.Println(err)
			return code.FailedExec
		}
		return code.OK
	}

	if args.Schema {
		ref := args.ValidateAgainst
		if ref == "" {
			mrt.WriteAbsoluteReferences(os.Stdout)
			return code.OK
		}

		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Println("[ERR]", err)
			return code.FailedSchema
		}

		if err := mrt.ValidateAgainstSchema(ref, data); err != nil {
			switch err {
			case modeler.ErrUnparsablePayload:
			case modeler.ErrNoSuchSchema:
				as.ColorERR.Printf("No such $ref '%s'\n", ref)
				mrt.WriteAbsoluteReferences(os.Stdout)
			default:
				as.ColorERR.Println(err)
			}
			return code.FailedSchema
		}
		as.ColorNFO.Println("Payload is valid")
		return code.OK
	}

	if args.Shrink != "" {
		// mrt.shrinking = true
		// mrt.unshrunk = len(toShrink)
		msg := "--shrink=ID isn't implemented yet."
		log.Println("[ERR]", msg)
		as.ColorERR.Println(msg)
		return code.Failed
	}

	apiKey := os.Getenv(envAPIKey)
	if apiKey == "" {
		err := fmt.Errorf("$%s is unset", envAPIKey)
		log.Println("[ERR]", err)
		as.ColorERR.Println(err)
		return code.Failed
	}

	as.ColorNFO.Printf("%d named schemas\n", mrt.InputsCount())
	if err = mrt.FilterEndpoints(os.Args); err != nil {
		as.ColorERR.Println(err)
		return code.Failed
	}

	as.ColorNFO.Printf("\n Running tests...\n\n")
	err = mrt.Fuzz(ctx, args.N, []byte(args.Seed), args.NoShrinking, apiKey)
	switch {
	case err == nil:
	case err == context.Canceled:
		as.ColorERR.Println("Testing interrupted.")
		return code.Failed
	case strings.Contains(err.Error(), context.DeadlineExceeded.Error()):
		as.ColorERR.Printf("Testing interrupted after %s.\n", args.OverallBudgetTime)
		return code.OK
	default:
		log.Println("[ERR]", err)
	}
	switch err.(type) {
	case *rt.TestingCampaignSuccess:
		return code.OK
	case *rt.TestingCampaignFailure:
		return code.FailedFuzz
	case *rt.TestingCampaignFailureDueToResetterError:
		as.ColorERR.Println(err)
		return code.FailedExec
	}
	defer as.ColorWRN.Println("You might want to run $", binName, "exec stop")
	return retryOrReport()
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

func doLogs(offset uint64) int {
	if err := cwid.MakePwdID(binName, offset); err != nil {
		return retryOrReport()
	}

	fn := cwid.LogFile()
	os.Stderr.WriteString(fn + "\n")
	f, err := os.Open(fn)
	if err != nil {
		as.ColorERR.Println(err)
		return code.Failed
	}
	defer f.Close()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		as.ColorERR.Println(err)
		return retryOrReport()
	}
	return code.OK
}

func doUpdate() int {
	rel := update.NewGithubRelease(githubSlug, binName)
	latest, err := rel.PeekLatestRelease()
	if err != nil {
		return retryOrReport()
	}

	if latest != binVersion {
		fmt.Println("A version newer than", binVersion, "is out:", latest)
		if err := rel.ReplaceCurrentRelease(latest); err != nil {
			as.ColorERR.Println("The update failed 🙈 please try again later")
			const latest = "https://github.com/" + githubSlug + "/releases/latest"
			fmt.Println("You can always upgrade from", latest)
			return code.FailedUpdate
		}
	}
	return code.OK
}

func doEnv(vars []string) int {
	all := map[string]bool{
		envAPIKey: false,
	}
	penv := func(key string) { fmt.Printf("%s=%q\n", key, os.Getenv(key)) }
	if len(vars) == 0 {
		for key := range all {
			penv(key)
		}
		return code.OK
	}

	for _, key := range vars {
		if printed, ok := all[key]; !ok || printed {
			return code.Failed
		}
		all[key] = true
		penv(key)
	}
	return code.OK
}

func retryOrReport() int {
	const issues = "https://github.com/" + githubSlug + "/issues"
	const email = "ook@fuzzymonkey.co"
	w := os.Stderr
	fmt.Fprintln(w, "\nLooks like something went wrong... Maybe try again with -vv?")
	fmt.Fprintf(w, "\nYou may want to try `monkey update`.\n")
	fmt.Fprintf(w, "\nIf that doesn't fix it, take a look at %s\n", cwid.LogFile())
	fmt.Fprintf(w, "or come by %s\n", issues)
	fmt.Fprintf(w, "or drop us a line at %s\n", email)
	fmt.Fprintln(w, "\nThank you for your patience & sorry about this :)")
	return code.Failed
}
