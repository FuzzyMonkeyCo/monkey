package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime/ctxvalues"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Fuzz TODO
func (rt *Runtime) Fuzz(ctx context.Context, ntensity uint32, apiKey string) (err error) {
	ctx = metadata.AppendToOutgoingContext(ctx,
		"ua", rt.binTitle,
		"apiKey", apiKey,
	)

	if rt.client, err = fm.NewCh(ctx); err != nil {
		return
	}
	defer rt.client.Close()

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}
	resetter := mdl.GetResetter()
	resetter.Env(rt.envRead)

	rt.newProgress(ctx, ntensity)
	// Pass user agent down to caller
	ctx = context.WithValue(ctx, ctxvalues.UserAgent, rt.binTitle)

	log.Printf("[DBG] sending initial msg")
	select {
	case <-time.After(txTimeout):
		err = errTXTimeout
	case err = <-rt.client.Snd(&fm.Clt{
		Msg: &fm.Clt_Fuzz_{
			Fuzz: &fm.Clt_Fuzz{
				EIDs:      rt.eIds,
				EnvRead:   rt.envRead,
				Model:     mdl.ToProto(),
				ModelKind: fm.Clt_Fuzz_OpenAPIv3,
				Ntensity:  ntensity,
				Resetter:  resetter.ToProto(),
				// FIXME: seeding
				Seed:  []byte{42, 42, 42},
				Tags:  rt.tags,
				Usage: os.Args,
			}}}):
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			log.Println("[ERR]", err)
		default:
		}
		if err != nil {
			break
		}

		log.Printf("[DBG] receiving msg...")
		var srv *fm.Srv
		select {
		case err = <-rt.client.RcvErr():
		case srv = <-rt.client.RcvMsg():
		case <-time.After(txTimeout):
			err = errTXTimeout
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			log.Println("[ERR]", err)
			break
		}

		switch srv.GetMsg().(type) {
		case *fm.Srv_Call_:
			log.Println("[NFO] handling fm.Srv_Call_")
			cll := srv.GetCall()
			if err = rt.call(ctx, cll); err != nil {
				break
			}
			log.Println("[NFO] done handling fm.Srv_Call_")
		case *fm.Srv_Reset_:
			log.Println("[NFO] handling fm.Srv_Reset_")
			if err = rt.reset(ctx); err != nil {
				break
			}
			log.Println("[NFO] done handling fm.Srv_Reset_")
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
		if err2 := rt.recvFuzzProgress(ctx); err2 != nil {
			log.Println("[ERR]", err2)
			if err == nil {
				err = err2
			}
			break
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
