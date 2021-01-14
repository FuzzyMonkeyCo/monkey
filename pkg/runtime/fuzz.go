package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime/ctxvalues"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Fuzz runs calls, resets and live reporting
func (rt *Runtime) Fuzz(
	ctx context.Context,
	ntensity uint32,
	seed []byte,
	noShrinking bool,
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
		Shrink:   rt.shrinking,
		EnvRead:  rt.envRead,
		Model:    mdl.ToProto(),
		Ntensity: ntensity,
		Resetter: resetter.ToProto(),
		Seed:     seed,
		Tags:     rt.tags,
		Usage:    os.Args,
		UUID:     uuid.New().String(),
	}}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

	var toShrink []uint32
	var nextSeed []byte
	for {
		log.Printf("[DBG] receiving msg...")
		var srv *fm.Srv
		if srv, err = rt.client.Receive(ctx); err != nil {
			log.Println("[ERR]", err)
			if err == io.EOF {
				// Remote hang up: we're probably finished here
				err = nil
			}
			break
		}

		func() {
			if rt.progress == nil {
				fuzzRep := srv.GetFuzzRep()
				if err = rt.newProgress(ctx, fuzzRep.GetMaxTestsCount()); err != nil {
					return
				}
				seed = fuzzRep.GetSeed()
				rt.progress.Printf("  --seed='%s'", seed)
				return
			}

			if fp := srv.GetFuzzingProgress(); fp != nil {
				rt.fuzzingProgress(fp)
				srv.FuzzingProgress = nil
			}

			msg := srv.GetMsg()
			log.Printf("[NFO] handling %T", msg)
			// Logical errors must not stop communication,
			// only transport errors should.
			switch msg := msg.(type) {
			// case nil: unreachable
			case *fm.Srv_Call_:
				if err = rt.call(ctx, msg.Call); err != nil {
					return
				}
			case *fm.Srv_Reset_:
				if err = rt.reset(ctx); err != nil {
					return
				}
			case *fm.Srv_FuzzingResult_:
				toShrink = msg.FuzzingResult.GetEIDs()
				nextSeed = msg.FuzzingResult.GetSeed()
				if rt.shrinkingTimes == nil {
					value := msg.FuzzingResult.GetMaxShrinks()
					rt.shrinkingTimes = &value
				}
				return
			default: // unreachable
				err = fmt.Errorf("unhandled srv msg %T: %+v", msg, srv)
				log.Println("[ERR]", err)
				return
			}
			log.Printf("[NFO] handled %T", msg)
		}()
		if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
			log.Println("[NFO] remote canceled campaign")
			break
		}
	}

	log.Println("[NFO] terminating resetter")
	if errR := mdl.GetResetter().Terminate(ctx, false); errR != nil {
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

	log.Println("[NFO] summing up test campaign")
	errS := rt.campaignSummary(toShrink, noShrinking, seed)
	log.Println("[ERR] campaignSummary", errS)

	if err != nil {
		// Cannot continue after transport or any termination error
		return
	}

	if _, ok := errS.(*TestingCampaignShrinkable); ok && !noShrinking {
		log.Println("[NFO] about to shrink that bug")
		if !rt.shrinking {
			rt.unshrunk = uint32(len(toShrink))
		}
		rt.shrinking = true
		rt.eIds = uniqueEIDs(toShrink)
		*rt.shrinkingTimes--
		return rt.Fuzz(ctx, ntensity, nextSeed, noShrinking, apiKey)
	}

	log.Println("[NFO] all finished up")
	return errS
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
