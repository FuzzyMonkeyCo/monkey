package runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/ci"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/cli"
)

func (rt *Runtime) newProgress(ctx context.Context, max uint32) {
	envSetAndNonEmpty := func(key string) bool {
		val, ok := os.LookupEnv(key)
		return ok && len(val) != 0
	}

	if rt.logLevel != 0 || envSetAndNonEmpty("CI") {
		rt.progress = &ci.Progresser{}
		if rt.logLevel == 0 {
			rt.logLevel = 3 // lowest level: DBG
		}
	} else {
		rt.progress = &cli.Progresser{}
	}
	rt.testingCampaingStart = time.Now()
	rt.progress.WithContext(ctx)
	rt.progress.MaxTestsCount(max)
}

func (rt *Runtime) recvFuzzingProgress(ctx context.Context) (err error) {
	log.Println("[DBG] receiving fm.Srv_FuzzingProgress...")
	var srv *fm.Srv
	if srv, err = rt.client.Receive(ctx); err != nil {
		log.Println("[ERR]", err)
		return
	}
	fp := srv.GetFuzzingProgress()
	if fp == nil {
		err = fmt.Errorf("empty Srv_FuzzingProgress: %+v", srv)
		log.Println("[ERR]", err)
		return
	}
	rt.fuzzingProgress(fp)
	return
}

func (rt *Runtime) fuzzingProgress(fp *fm.Srv_FuzzingProgress) {
	log.Println("[DBG] srvprogress:", fp)
	rt.progress.TotalTestsCount(fp.GetTotalTestsCount())
	rt.progress.TotalCallsCount(fp.GetTotalCallsCount())
	rt.progress.TotalChecksCount(fp.GetTotalChecksCount())
	rt.progress.TestCallsCount(fp.GetTestCallsCount())
	rt.progress.CallChecksCount(fp.GetCallChecksCount())
	rt.lastFuzzingProgress = fp
}

// TestingCampaingOutcomer describes a testing campaing's results
type TestingCampaingOutcomer interface {
	error
	isTestingCampaingOutcomer()
}

var _ TestingCampaingOutcomer = (*TestingCampaingSuccess)(nil)
var _ TestingCampaingOutcomer = (*TestingCampaingFailure)(nil)

// TestingCampaingSuccess indicates no bug was found during fuzzing.
type TestingCampaingSuccess struct{}

// TestingCampaingFailure indicates a bug was found during fuzzing.
type TestingCampaingFailure struct{}

func (tc *TestingCampaingSuccess) Error() string { return "Found no bug" }
func (tc *TestingCampaingFailure) Error() string { return "Found a bug" }

func (tc *TestingCampaingSuccess) isTestingCampaingOutcomer() {}
func (tc *TestingCampaingFailure) isTestingCampaingOutcomer() {}

// campaignSummary concludes the testing campaing and reports to the user.
func (rt *Runtime) campaignSummary() TestingCampaingOutcomer {
	l := rt.lastFuzzingProgress
	fmt.Println()
	fmt.Println()
	as.ColorWRN.Println(
		"Ran", l.GetTotalTestsCount(), plural("test", l.GetTotalTestsCount()),
		"totalling", l.GetTotalCallsCount(), plural("request", l.GetTotalCallsCount()),
		"and", l.GetTotalChecksCount(), plural("check", l.GetTotalChecksCount()),
		"in", time.Since(rt.testingCampaingStart))

	if l.GetSuccess() {
		as.ColorNFO.Println("No bugs found... yet.")
		return &TestingCampaingSuccess{}
	}

	if l.GetTestCallsCount() == 0 {
		as.ColorERR.Println("Something went wrong while resetting the system to a neutral state.")
		as.ColorNFO.Println("No bugs found... yet.")
		return &TestingCampaingFailure{}
	}

	as.ColorERR.Printf("A bug reproducible in %d HTTP %s was detected after %d",
		l.GetTestCallsCount(), plural("request", l.GetTestCallsCount()), l.GetTotalTestsCount())
	var m uint32 // FIXME: handle shrinking report
	switch {
	case l.GetTotalTestsCount() == 1:
		as.ColorERR.Printf(" %s.\n", plural("test", l.GetTotalTestsCount()))
	case m == 0:
		as.ColorERR.Printf(" %s and not yet shrunk.\n", plural("test", l.GetTotalTestsCount()))
		//TODO: suggest shrinking invocation
		// A task that tries to minimize a testcase to its smallest possible size, such that it still triggers the same underlying bug on the target program.
	case m == 1:
		as.ColorERR.Printf(" %s then shrunk once.\n", plural("test", l.GetTotalTestsCount()))
	default:
		as.ColorERR.Printf(" %s then shrunk %d %s.\n", plural("test", l.GetTotalTestsCount()), m, plural("time", m))
	}
	return &TestingCampaingFailure{}
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
