package runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime/ctxvalues"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// PastSeedMagic is a magic string to search through logs
const PastSeedMagic = "kFPoyHOKs6XrK2F7jPMGc51f@k&&@9T6LE!zn&uy"

// Fuzz runs calls, resets and live reporting
func (rt *Runtime) Fuzz(
	ctx context.Context,
	ntensity uint32,
	seed []byte,
	vvv uint8,
	ptype, apiKey string,
) (err error) {
	start := time.Now()
	if zeroTime := (time.Time{}); rt.fuzzingStartedAt == zeroTime {
		rt.fuzzingStartedAt = start
	}

	if apiKey != "" {
		// Pass user agent down to caller
		ctx = context.WithValue(ctx, ctxvalues.UserAgent, rt.binTitle)
		ctx = metadata.AppendToOutgoingContext(ctx,
			"ua", rt.binTitle,
			"apiKey", apiKey,
		)
	}
	if rt.client, err = fm.NewChBiDi(ctx); err != nil {
		return
	}
	defer rt.client.Close()

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}
	rsttr := mdl.GetResetter()
	rsttr.Env(rt.envRead)

	log.Printf("[DBG] sending initial msg")
	if err = rt.client.Send(ctx, &fm.Clt{Msg: &fm.Clt_Fuzz_{Fuzz: &fm.Clt_Fuzz{
		EIDs:     rt.eIds,
		EnvRead:  rt.envRead,
		Model:    mdl.ToProto(),
		Ntensity: ntensity,
		Resetter: rsttr.ToProto(),
		Seed:     seed,
		Labels:   rt.labels,
		Usage:    os.Args,
		UUIDs:    []string{uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()},
	}}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

	var result *fm.Srv_FuzzingResult
	suggestedSeed := seed
	for {
		log.Printf("[DBG] receiving msg...")
		var srv *fm.Srv
		if srv, err = rt.client.Receive(ctx); err != nil {
			log.Println("[ERR]", err)
			break
		}

		func() {
			if rt.progress == nil {
				fuzzRep := srv.GetFuzzRep()
				if err = rt.newProgress(ctx, fuzzRep.GetMaxTestsCount(), vvv, ptype); err != nil {
					return
				}
				if tkn := fuzzRep.GetToken(); tkn != "" {
					ctx = metadata.AppendToOutgoingContext(ctx, "token", fuzzRep.GetToken())
				}
				// Keep in this order (suggested last) for pastseed
				log.Printf("[ERR] (not an error) %s=%s (seed)", PastSeedMagic, fuzzRep.GetSeed())
				log.Printf("[ERR] (not an error) %s=%s (suggested)", PastSeedMagic, suggestedSeed)
				rt.progress.Printf("  --seed=%s", fuzzRep.GetSeed())
				return
			}

			if fp := srv.GetFuzzingProgress(); fp != nil {
				rt.fuzzingProgress(fp)
				srv.FuzzingProgress = nil
			}

			msg := srv.GetMsg()
			log.Printf("[NFO] handling %T", msg)
			defer log.Printf("[NFO] handled %T", msg)
			// Only transport errors should stop communication, not logical ones.
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
				result = msg.FuzzingResult
				suggestedSeed = result.GetSuggestedSeed()
				log.Printf("[ERR] (not an error) %s=%s (suggested)", PastSeedMagic, suggestedSeed)
				return
			default: // unreachable
				err = fmt.Errorf("unhandled srv msg %T: %+v", msg, srv)
				log.Println("[ERR]", err)
				return
			}
		}()
		if err != nil || result != nil {
			break
		}
	}

	if rt.progress != nil {
		// It is possible to receive an error as the first response
		// in which case rt.progress would be nil.
		log.Println("[NFO] terminating progresser")
		if errP := rt.progress.Terminate(); errP != nil {
			log.Println("[ERR]", errP)
			if err == nil {
				err = errP
			}
		}
		rt.progress = nil
	}

	l := rt.lastFuzzingProgress

	log.Printf("[NFO] ran %d tests: %d calls: %d checks",
		l.GetTotalTestsCount(), l.GetTotalCallsCount(), l.GetTotalChecksCount())
	as.ColorWRN.Printf("\n\nRan %d %s totalling %d %s and %d %s in %s.\n\n",
		l.GetTotalTestsCount(), plural("test", l.GetTotalTestsCount()),
		l.GetTotalCallsCount(), plural("call", l.GetTotalCallsCount()),
		l.GetTotalChecksCount(), plural("check", l.GetTotalChecksCount()),
		time.Since(start),
	)

	if err != nil {
		// Cannot continue after transport or any termination error
		return
	}

	if counterexample := result.GetCounterexample(); len(counterexample) != 0 {
		as.ColorNFO.Printf("Initial State")
		if kvs := rt.modelState0.Items(); len(kvs) == 0 {
			fmt.Println(" (empty)")
		} else {
			as.ColorNFO.Println(":")
			printModelState(kvs, as.ColorOK.Printf, 0)
		}
		as.ColorNFO.Println()

		as.ColorNFO.Printf("A test produced a bug in %d calls:\n", len(counterexample))
		as.ColorOK.Println("monkey exec start")
		for _, ceItem := range counterexample {
			as.ColorOK.Println(ceItem.CLIString())
		}
		as.ColorOK.Println("monkey exec stop")
		as.ColorNFO.Println()

		as.ColorNFO.Printf("Final State")
		if kvs := rt.modelState.Items(); len(kvs) == 0 {
			fmt.Println(" (empty)")
		} else {
			as.ColorNFO.Println(":")
			printModelState(kvs, as.ColorOK.Printf, 0)
		}
		as.ColorNFO.Println()
	}

	if result.GetWillNowShrink() {
		as.ColorNFO.Println()
		as.ColorNFO.Println("Shrinking...")
	}

	if newSeed := result.GetNextSeed(); len(newSeed) != 0 {
		log.Println("[NFO] continuing with new seed")
		return rt.Fuzz(ctx, ntensity, newSeed, vvv, ptype, "")
	}

	if l.GetSuccess() {
		as.ColorNFO.Println("No bugs found yet.")
		return &TestingCampaignSuccess{}
	}

	if l.GetTestCallsCount() == 0 {
		return &TestingCampaignFailureDueToResetterError{}
	}

	log.Printf("[NFO] found a bug in %d calls (while shrinking? %v)",
		l.GetTestCallsCount(), result.GetWasShrinking())
	as.ColorWRN.Printf("You should be able to reproduce this test failure with this flag:\n")
	as.ColorWRN.Printf("  --seed=%s\n", suggestedSeed)
	return &TestingCampaignFailure{}
}
