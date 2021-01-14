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
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/dots"
)

func (rt *Runtime) newProgress(ctx context.Context, max uint32) (err error) {
	envSetAndNonEmpty := func(key string) bool {
		val, ok := os.LookupEnv(key)
		return ok && val != ""
	}

	if rt.ptype == "" || rt.ptype == "auto" {
		if rt.logLevel != 0 || envSetAndNonEmpty("CI") {
			rt.ptype = "ci"
		} else {
			rt.ptype = "cli"
		}
	}

	switch rt.ptype {
	case "cli":
		rt.progress = &cli.Progresser{}
	case "ci":
		rt.progress = &ci.Progresser{}
		if rt.logLevel == 0 {
			rt.logLevel = 3 // lowest level: DBG
		}
	case "dots":
		rt.progress = &dots.Progresser{}
	default:
		err = fmt.Errorf("unexpected progresser %q", rt.ptype)
		log.Println("[ERR]", err)
		return
	}
	rt.testingCampaignStart = time.Now()
	rt.progress.WithContext(ctx)
	rt.progress.MaxTestsCount(max)
	return
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

// TestingCampaignOutcomer describes a testing campaign's results
type TestingCampaignOutcomer interface {
	error
	isTestingCampaignOutcomer()
}

var _ TestingCampaignOutcomer = (*TestingCampaignSuccess)(nil)
var _ TestingCampaignOutcomer = (*TestingCampaignFailure)(nil)
var _ TestingCampaignOutcomer = (*TestingCampaignShrinkable)(nil)
var _ TestingCampaignOutcomer = (*TestingCampaignFailureDueToResetterError)(nil)

// TestingCampaignSuccess indicates no bug was found during fuzzing.
type TestingCampaignSuccess struct{}

// TestingCampaignFailure indicates a bug was found during fuzzing.
type TestingCampaignFailure struct{}

// TestingCampaignShrinkable indicates a bug-producing test can be shrunk.
type TestingCampaignShrinkable struct{}

// TestingCampaignFailureDueToResetterError indicates a bug was found during reset.
type TestingCampaignFailureDueToResetterError struct{}

func (tc *TestingCampaignSuccess) Error() string    { return "Found no bug" }
func (tc *TestingCampaignFailure) Error() string    { return "Found a bug" }
func (tc *TestingCampaignShrinkable) Error() string { return "Found a shrinkable bug" }
func (tc *TestingCampaignFailureDueToResetterError) Error() string {
	return "Something went wrong while resetting the system to a neutral state."
}

func (tc *TestingCampaignSuccess) isTestingCampaignOutcomer()                   {}
func (tc *TestingCampaignFailure) isTestingCampaignOutcomer()                   {}
func (tc *TestingCampaignShrinkable) isTestingCampaignOutcomer()                {}
func (tc *TestingCampaignFailureDueToResetterError) isTestingCampaignOutcomer() {}

// campaignSummary concludes the testing campaign and reports to the user.
func (rt *Runtime) campaignSummary(
	shrinkable []uint32,
	noShrinking bool,
	seed []byte,
) TestingCampaignOutcomer {
	l := rt.lastFuzzingProgress
	log.Printf("[NFO] ran %d tests: %d calls: %d checks",
		l.GetTotalTestsCount(), l.GetTotalCallsCount(), l.GetTotalChecksCount())
	as.ColorWRN.Printf("\n\nRan %d %s totalling %d %s and %d %s in %s.\n",
		l.GetTotalTestsCount(), plural("test", l.GetTotalTestsCount()),
		l.GetTotalCallsCount(), plural("call", l.GetTotalCallsCount()),
		l.GetTotalChecksCount(), plural("check", l.GetTotalChecksCount()),
		time.Since(rt.testingCampaignStart),
	)

	if l.GetSuccess() {
		as.ColorNFO.Println("No bugs found yet.")
		return &TestingCampaignSuccess{}
	}

	if l.GetTestCallsCount() == 0 {
		return &TestingCampaignFailureDueToResetterError{}
	}

	log.Printf("[NFO] found a bug in %d calls: %+v (shrinking? %v)",
		l.GetTestCallsCount(), rt.eIds, rt.shrinking)
	as.ColorERR.Printf("A bug was detected after %d %s.\n",
		l.GetTestCallsCount(), plural("call", l.GetTestCallsCount()),
	)

	if !noShrinking && rt.shrinkingTimes != nil && *rt.shrinkingTimes != 0 && len(shrinkable) != 0 {
		as.ColorNFO.Printf("Trying to reproduce this bug in fewer than %d %s...\n",
			l.GetTestCallsCount(), plural("call", l.GetTestCallsCount()))
		return &TestingCampaignShrinkable{}
	}

	if rt.shrinking {
		// if l.GetTestCallsCount() == rt.unshrunk {
		// 	as.ColorNFO.Println("Shrinking done.")
		// } else {
		as.ColorNFO.Printf("Before shrinking, it took %d %s to produce a bug.\n",
			rt.unshrunk, plural("call", rt.unshrunk))
		// }
	}
	as.ColorWRN.Printf("You can try to reproduce this test failure with this flag:\n  --seed='%s'\n", seed)

	return &TestingCampaignFailure{}
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

// func equalEIDs(a, b []uint32) bool {
// 	if len(a) != len(b) {
// 		return false
// 	}
// 	for i, x := range a {
// 		if x != b[i] {
// 			return false
// 		}
// 	}
// 	return true
// }
