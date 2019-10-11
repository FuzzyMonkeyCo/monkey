package pkg

import (
	"context"
	"fmt"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

func (rt *runtime) Dial(ctx context.Context, ua, apiKey string) (
	closer func() error,
	err error,
) {
	if rt.clt, closer, err = fm.NewClient(ctx); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (rt *runtime) Fuzz(ctx context.Context) error {
	defer func() {
		if err := rt.clt.CloseSend(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	log.Printf("[DBG] 🡱  initial msg...")
	if err := rt.clt.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_Fuzz_{
				Fuzz: &fm.Clt_Msg_Fuzz{
					Resetter:  rt.modelers[0].GetSUTResetter().ToProto(),
					ModelKind: "OpenAPIv3",
					Model:     rt.modelers[0].ToProto(),
					Usage:     os.Args,
					Seed:      []byte{42, 42, 42},
					Intensity: rt.Ntensity,
					EIDs:      rt.EIDs,
				}}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

	for {
		srv, err := stream.Recv()
		if err == io.EOF {
			if err := rt.modelers[0].GetSUTResetter().Terminate(ctx, nil); err != nil {
				return err
			}
			if err := rt.progress.Terminate(); err != nil {
				return err
			}
			return nil
			mnk.progress.bar.Done()
		}
		if err != nil {
			log.Println("[ERR]", err)
			return err
		}

		log.Println(srv)
		switch msg := srv.GetMsg().(type) {
		case *fm.Srv_Msg_Call:
		case *fm.Srv_Msg_Reset:
			if err := rt.reset(); err != nil {
				return err
			}
		default:
			err := fmt.Errorf("unhandled srv msg %T: %+v", msg, msg)
			log.Println("[ERR]", err)
			return err
		}
	}
}

// 		case *Msg_FuzzProgress:
// 			done := msg.GetFuzzProgress()
// 			if err = done.exec(mnk); err != nil {
// 				return
// 			}
// 			if done.GetFailure() || done.GetSuccess() {
// 				return
// 			}

// mnk.progress = newProgress(mnk.Cfg.N)
