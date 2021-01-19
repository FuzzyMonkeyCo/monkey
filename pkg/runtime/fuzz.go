package runtime

import (
	"context"
	"fmt"
	// "io"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime/ctxvalues"
	"github.com/google/uuid"
	// "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	// "google.golang.org/grpc/status"
)

// Fuzz runs calls, resets and live reporting
func (rt *Runtime) Fuzz(
	ctx context.Context,
	ntensity uint32,
	seed []byte,
	// noShrinking bool,
	apiKey string,
) (err error) {
	ctx = metadata.AppendToOutgoingContext(ctx,
		"ua", rt.binTitle,
		"apiKey", apiKey,
	)

	if rt.client, err = fm.NewChBiDi(ctx); err != nil {
		return
	}
	defer rt.client.Close()

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}
	resetter := mdl.GetResetter()
	resetter.Env(rt.envRead)

	// Pass user agent down to caller
	ctx = context.WithValue(ctx, ctxvalues.UserAgent, rt.binTitle)

	log.Printf("[DBG] sending initial msg")
	if err = rt.client.Send(ctx, &fm.Clt{Msg: &fm.Clt_Fuzz_{Fuzz: &fm.Clt_Fuzz{
		EIDs:     rt.eIds,
		EnvRead:  rt.envRead,
		Model:    mdl.ToProto(),
		Ntensity: ntensity,
		Resetter: resetter.ToProto(),
		Seed:     seed,
		Tags:     rt.tags,
		Usage:    os.Args,
		UUIDs:    []string{uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()},
	}}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

	var result *fm.Srv_FuzzingResult
	// var toShrink []uint32
	// var nextSeed []byte
	for {
		log.Printf("[DBG] receiving msg...")
		var srv *fm.Srv
		if srv, err = rt.client.Receive(ctx); err != nil {
			log.Println("[ERR]", err)
			// if err == io.EOF &&result!=nil{
			// 	// Remote hang up: we're probably finished here
			// 	err = nil
			// }
			break
		}

		func() {
			if rt.progress == nil {
				fuzzRep := srv.GetFuzzRep()
				if err = rt.newProgress(ctx, fuzzRep.GetMaxTestsCount()); err != nil {
					return
				}
				if fuzzRep.GetIsShrinking() {
					rt.progress.Printf("Shrinking...\n")
				}
				rt.progress.Printf("  --seed='%s'", fuzzRep.GetSeed())
				return
			}

			if fp := srv.GetFuzzingProgress(); fp != nil {
				rt.fuzzingProgress(fp)
				srv.FuzzingProgress = nil
			}

			msg := srv.GetMsg()
			log.Printf("[NFO] handling %T", msg)
			defer log.Printf("[NFO] handled %T", msg)
			// Logical errors must not stop communication,
			// only transport errors should.
			switch msg := msg.(type) {
			case nil:
				return
			case *fm.Srv_Call_:
				if err = rt.call(ctx, msg.Call); err != nil {
					return
				}
			case *fm.Srv_Reset_:
				if err = rt.reset(ctx); err != nil {
					return
				}
			case *fm.Srv_FuzzingResult_:
				// toShrink = msg.FuzzingResult.GetEIDs()
				// nextSeed = msg.FuzzingResult.GetSeed()
				// if rt.shrinkingTimes == nil {
				// 	value := msg.FuzzingResult.GetMaxShrinks()
				// 	rt.shrinkingTimes = &value
				// }
				result = msg.FuzzingResult
				return
			default: // unreachable
				err = fmt.Errorf("unhandled srv msg %T: %+v", msg, srv)
				log.Println("[ERR]", err)
				return
			}
		}()
		// if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
		// 	log.Println("[NFO] remote canceled campaign")
		// 	break
		// }
		if err != nil || result != nil {
			break
		}
	}

	log.Println("[NFO] terminating resetter")
	if errR := mdl.GetResetter().Terminate(ctx); errR != nil {
		log.Println("[ERR]", errR)
		if err == nil {
			err = errR
		}
	}
	log.Println("[NFO] terminating progresser")
	if errP := rt.progress.Terminate(); errP != nil {
		log.Println("[ERR]", errP)
		if err == nil {
			err = errP
		}
	}

	rt.progress = nil

	if l := rt.lastFuzzingProgress; true {
		log.Printf("[NFO] ran %d tests: %d calls: %d checks",
			l.GetTotalTestsCount(), l.GetTotalCallsCount(), l.GetTotalChecksCount())
		as.ColorWRN.Printf("\n\nRan %d %s totalling %d %s and %d %s in %s.\n",
			l.GetTotalTestsCount(), plural("test", l.GetTotalTestsCount()),
			l.GetTotalCallsCount(), plural("call", l.GetTotalCallsCount()),
			l.GetTotalChecksCount(), plural("check", l.GetTotalChecksCount()),
			time.Since(rt.testingCampaignStart),
		)
	}

	if err != nil {
		// Cannot continue after transport or any termination error
		return
	}

	// if result == nil {
	// 	panic(">>> nil result => check err first (e.g context deadline exceeded)")
	// 	// check err but first do show
	// 	//Ran 7670 tests totalling 61353 calls and 436953 checks in 29m56.745711691s.
	// 	// then see with main's
	// 	//Testing interrupted after --time-budget-overall=30m0s.
	// 	///basically inline summaryFrom here
	// }
	// errS := rt.summaryFrom(result, err)
	// // errS := rt.campaignSummary(toShrink, noShrinking, seed)
	// log.Println("[ERR] campaignSummary", errS)

	// // if err != nil {
	// // 	return
	// // }

	// if _, ok := errS.(*TestingCampaignShrinkable); ok && !noShrinking {
	// 	log.Println("[NFO] about to shrink that bug")
	// 	if !rt.shrinking {
	// 		rt.unshrunk = uint32(len(toShrink))
	// 	}
	// 	rt.shrinking = true
	// 	rt.eIds = uniqueEIDs(toShrink)
	// 	*rt.shrinkingTimes--GetSeedUsed
	// 	return rt.Fuzz(ctx, ntensity, nextSeed, noShrinking, apiKey)
	// }
	// ^ move from MNK:
	// * keep only loop -> break on result
	// * SRV must decide whether to continue after failure/shrink
	// * send noShrinking (.Usage is already sent)
	// * do not send failing eids: SRV can work this out
	// * receive nextSeed but try not to parse whole spec again (opti)
	// if _, ok := errS.(*TestingCampaignShrinkable); ok {
	// 	log.Println("[NFO] continuing with new seed")
	// }
	if newSeed := result.GetNextSeed(); len(newSeed) != 0 {
		log.Println("[NFO] continuing with new seed")
		return rt.Fuzz(ctx, ntensity, newSeed, apiKey)
	}

	l := rt.lastFuzzingProgress

	if l.GetSuccess() {
		as.ColorNFO.Println("No bugs found yet.")
		return &TestingCampaignSuccess{}
	}

	if l.GetTestCallsCount() == 0 {
		return &TestingCampaignFailureDueToResetterError{}
	}

	log.Printf("[NFO] found a bug in %d calls (while shrinking? %v)",
		l.GetTestCallsCount(), result.GetWasShrinking())
	as.ColorERR.Printf("A bug was detected after %d %s.\n",
		l.GetTestCallsCount(), plural("call", l.GetTestCallsCount()),
	)

	if result.GetWillNowShrink() {
		as.ColorNFO.Printf("Now trying to minimize this bug...\n")
		// // if !noShrinking && rt.shrinkingTimes != nil && *rt.shrinkingTimes != 0 && len(shrinkable) != 0 {
		// as.ColorNFO.Printf("Trying to reproduce this bug in fewer than %d %s...\n",
		// 	l.GetTestCallsCount(), plural("call", l.GetTestCallsCount()))
		// return &TestingCampaignShrinkable{}
	}

	// if rt.shrinking {
	// 	// if l.GetTestCallsCount() == rt.unshrunk {
	// 	// 	as.ColorNFO.Println("Shrinking done.")
	// 	// } else {
	// 	as.ColorNFO.Printf("Before shrinking, it took %d %s to produce a bug.\n",
	// 		rt.unshrunk, plural("call", rt.unshrunk))
	// 	// }
	// }
	as.ColorWRN.Printf("You can try to reproduce the test failure with this flag:\n")
	as.ColorWRN.Printf("  --seed='%s'\n", result.GetSeedUsed())

	// Ran 7670 tests totalling 61353 calls and 436953 checks in 29m56.745711691s.
	// A bug was detected after 1 call.
	// You can try to reproduce the test failure with this flag:
	//   --seed=''
	// Testing interrupted after --time-budget-overall=30m0s.
	// bug1_modified fuzzymonkey__start_reset_stop.star V=0 T=6 (got 0) ...failed
	// + echo Stopping...
	// + RELX_REPLACE_OS_VARS=true
	// + ./_build/prod/rel/sample/bin/sample stop
	// Stopping...

	return &TestingCampaignFailure{}
}

func uniqueEIDs(EIDs []uint32) []uint32 {
	set := make(map[uint32]struct{}, len(EIDs))
	for _, EID := range EIDs {
		set[EID] = struct{}{}
	}
	unique := make([]uint32, 0, len(set))
	for EID := range set {
		unique = append(unique, EID)
	}
	return unique
}
