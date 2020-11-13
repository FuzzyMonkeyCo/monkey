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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Fuzz runs calls, resets and live reporting
func (rt *Runtime) Fuzz(ctx context.Context, ntensity uint32, apiKey string) (err error) {
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
		Seed:     []byte{42, 42, 42}, //FIXME
		Tags:     rt.tags,
		Usage:    os.Args,
	}}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

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

		if rt.progress == nil {
			rt.newProgress(ctx, srv.GetFuzzRep().GetMaxTestsCount())
		}

		if fp := srv.GetFuzzingProgress(); fp != nil {
			rt.fuzzingProgress(fp)
		}

		switch srv.GetMsg().(type) {
		case nil:
		case *fm.Srv_Call_:
			log.Println("[NFO] handling fm.Srv_Call_")
			cll := srv.GetCall()
			if err = rt.call(ctx, cll); err != nil {
				break
			}
			log.Println("[NFO] handled fm.Srv_Call_")
		case *fm.Srv_Reset_:
			log.Println("[NFO] handling fm.Srv_Reset_")
			if err = rt.reset(ctx); err != nil {
				break
			}
			log.Println("[NFO] handled fm.Srv_Reset_")
		case *fm.Srv_FuzzingResult_: // FIXME
		default:
			err = fmt.Errorf("unhandled srv msg %T: %+v", srv.GetMsg(), srv)
			log.Println("[ERR]", err)
			break
		}
		if err != nil {
			if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
				log.Println("[NFO] got canceled...")
				break
			}
		}
	}

	log.Println("[DBG] server dialog ended, cleaning up...")
	log.Println("[NFO] terminating resetter")
	if err2 := mdl.GetResetter().Terminate(ctx, false); err2 != nil {
		log.Println("[ERR]", err2)
		if err == nil {
			err = err2
		}
	}
	log.Println("[NFO] terminating progresser")
	if err2 := rt.progress.Terminate(); err2 != nil {
		log.Println("[ERR]", err2)
		if err == nil {
			err = err2
		}
	}

	log.Println("[NFO] all finished up")
	if err == nil || err == modeler.ErrCheckFailed {
		err = rt.campaignSummary()
	}
	return
}
