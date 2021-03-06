package runtime

import (
	"context"
	"fmt"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/bar"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/ci"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/dots"
)

func (rt *Runtime) newProgress(ctx context.Context, max uint32, vvv uint8, ptype string) (err error) {
	if ptype == "" {
		if vvv != 0 {
			ptype = "ci"
		} else {
			ptype = "dots"
		}
		log.Printf("[NFO] using --progress=%s", ptype)
	}
	switch ptype {
	case "bar":
		rt.progress = &bar.Progresser{}
	case "ci":
		rt.progress = &ci.Progresser{}
		if vvv == 0 {
			vvv = 3 // lowest level: DBG
		}
	case "dots":
		rt.progress = &dots.Progresser{}
	default:
		err = fmt.Errorf("unexpected progresser %q", ptype)
		log.Println("[ERR]", err)
		return
	}
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
