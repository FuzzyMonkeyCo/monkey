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
var _ TestingCampaingOutcomer = (*TestingCampaingShrinkable)(nil)

// TestingCampaingSuccess indicates no bug was found during fuzzing.
type TestingCampaingSuccess struct{}

// TestingCampaingFailure indicates a bug was found during fuzzing.
type TestingCampaingFailure struct{}

// TestingCampaingShrinkable indicates a bug-producing test can be shrunk.
type TestingCampaingShrinkable struct{}

func (tc *TestingCampaingSuccess) Error() string    { return "Found no bug" }
func (tc *TestingCampaingFailure) Error() string    { return "Found a bug" }
func (tc *TestingCampaingShrinkable) Error() string { return "Found a bug" }

func (tc *TestingCampaingSuccess) isTestingCampaingOutcomer()    {}
func (tc *TestingCampaingFailure) isTestingCampaingOutcomer()    {}
func (tc *TestingCampaingShrinkable) isTestingCampaingOutcomer() {}

// campaignSummary concludes the testing campaing and reports to the user.
func (rt *Runtime) campaignSummary(in, shrinkable []uint32) TestingCampaingOutcomer {
	l := rt.lastFuzzingProgress
	log.Printf("[NFO] ran %d tests: %d calls: %d checks",
		l.GetTotalTestsCount(), l.GetTotalCallsCount(), l.GetTotalChecksCount())
	as.ColorWRN.Printf("\n\nRan %d %s totalling %d %s and %d %s in %s.\n",
		l.GetTotalTestsCount(), plural("test", l.GetTotalTestsCount()),
		l.GetTotalCallsCount(), plural("call", l.GetTotalCallsCount()),
		l.GetTotalChecksCount(), plural("check", l.GetTotalChecksCount()),
		time.Since(rt.testingCampaingStart),
	)

	if l.GetSuccess() {
		as.ColorNFO.Println("No bugs found yet.")
		return &TestingCampaingSuccess{}
	}

	if l.GetTestCallsCount() == 0 {
		as.ColorERR.Println("Something went wrong while resetting the system to a neutral state.")
		as.ColorNFO.Println("No bugs found yet.")
		return &TestingCampaingFailure{}
	}

	log.Printf("[NFO] found a bug in %d calls: %+v (shrinking? %v)",
		l.GetTestCallsCount(), in, rt.shrinking)
	as.ColorERR.Printf("A bug was detected after %d %s.\n",
		l.GetTestCallsCount(), plural("call", l.GetTestCallsCount()),
	)

	if len(shrinkable) != 0 && !equalEIDs(in, shrinkable) {
		as.ColorNFO.Printf("Trying to reproduce this bug in less than %d %s...\n",
			l.GetTestCallsCount(), plural("call", l.GetTestCallsCount()))
		return &TestingCampaingShrinkable{}
	}

	if rt.shrinking {
		as.ColorNFO.Printf("Before shrinking, it took %d %s to produce a bug.\n",
			rt.unshrunk, plural("call", rt.unshrunk))
	}

	return &TestingCampaingFailure{}
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

func equalEIDs(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if x != b[i] {
			return false
		}
	}
	return true
}
