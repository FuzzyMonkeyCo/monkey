package house

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

const tx30sTimeout = 30 * time.Second

var err30sTimeout = errors.New("gRPC recv timeout after 30s")

func (rt *Runtime) recvFuzzProgress(ctx context.Context) (err error) {
	log.Println("[DBG] receiving fm.Srv_FuzzProgress_...")
	var srv *fm.Srv
	select {
	case <-time.After(tx30sTimeout):
		err = err30sTimeout
	case srv = <-rt.client.RcvMsg():
	case err = <-rt.client.RcvErr():
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	switch srv.GetMsg().(type) {
	case *fm.Srv_FuzzProgress_:
		log.Println("[NFO] handling srvprogress")
		stts := srv.GetFuzzProgress()
		log.Println("[DBG] srvprogress:", stts)
		rt.progress.TotalTestsCount(stts.GetTotalTestsCount())
		rt.progress.TotalCallsCount(stts.GetTotalCallsCount())
		rt.progress.TotalChecksCount(stts.GetTotalChecksCount())
		rt.progress.TestCallsCount(stts.GetTestCallsCount())
		rt.progress.CallChecksCount(stts.GetCallChecksCount())
		rt.lastFuzzProgress = stts
		log.Println("[NFO] done handling srvprogress")
		return
	default:
		err = fmt.Errorf("unexpected srv msg %T: %+v", srv.GetMsg(), srv)
		log.Println("[ERR]", err)
		return
	}
}

type testingCampaingOutcomer interface {
	error
	isTestingCampaingOutcomer()
}

var _ testingCampaingOutcomer = (*TestingCampaingSuccess)(nil)
var _ testingCampaingOutcomer = (*TestingCampaingFailure)(nil)

type TestingCampaingSuccess struct{}
type TestingCampaingFailure struct{}

func (_ *TestingCampaingSuccess) Error() string { return "Found no bug" }
func (_ *TestingCampaingFailure) Error() string { return "Found a bug" }

func (_ *TestingCampaingSuccess) isTestingCampaingOutcomer() {}
func (_ *TestingCampaingFailure) isTestingCampaingOutcomer() {}

// campaignSummary concludes the testing campaing and reports to the user.
func (rt *Runtime) campaignSummary() testingCampaingOutcomer {
	l := rt.lastFuzzProgress
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
