package house

import (
	"fmt"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

func (rt *Runtime) recvFuzzProgress() error {
	log.Println("[DBG] receiving fm.Srv_FuzzProgress_...")
	srv, err := rt.client.Recv()
	if err != nil {
		log.Println("[ERR]", err)
		return err
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
		if stts.GetSuccess() {
			rt.progress.CampaignSuccess(true)
		} else if stts.GetFailure() {
			rt.progress.CampaignSuccess(false)
		}
		log.Println("[NFO] done handling srvprogress")
		return nil
	default:
		err := fmt.Errorf("unexpected srv msg %T: %+v", srv.GetMsg(), srv)
		log.Println("[ERR]", err)
		return err
	}
}
