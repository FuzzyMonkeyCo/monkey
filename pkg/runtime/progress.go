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

func (rt *Runtime) newProgress(ctx context.Context, ntensity uint32) {
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
	rt.progress.MaxTestsCount(10 * ntensity) // FIXME: learn magic number from Srv
}

func (rt *Runtime) recvFuzzingProgress(ctx context.Context) (err error) {
	log.Println("[DBG] receiving fm.Srv_FuzzingProgress_...")
	var srv *fm.Srv
	if srv, err = rt.client.Receive(ctx); err != nil {
		log.Println("[ERR]", err)
		return
	}

	switch srv.GetMsg().(type) {
	case *fm.Srv_FuzzingProgress_:
		log.Println("[NFO] handling srvprogress")
		stts := srv.GetFuzzingProgress()
		log.Println("[DBG] srvprogress:", stts)
		rt.progress.TotalTestsCount(stts.GetTotalTestsCount())
		rt.progress.TotalCallsCount(stts.GetTotalCallsCount())
		rt.progress.TotalChecksCount(stts.GetTotalChecksCount())
		rt.progress.TestCallsCount(stts.GetTestCallsCount())
		rt.progress.CallChecksCount(stts.GetCallChecksCount())
		rt.lastFuzzingProgress = stts
		log.Println("[NFO] done handling srvprogress")
		return
	default:
		err = fmt.Errorf("unexpected srv msg %T: %+v", srv.GetMsg(), srv)
		log.Println("[ERR]", err)
		return
	}
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
