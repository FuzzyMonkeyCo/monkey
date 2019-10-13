package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
)

func (rt *runtime) Dial(ctx context.Context, ua, apiKey string) (
	closer func() error,
	err error,
) {
	if rt.client, closer, err = fm.NewClient(ctx, ua, apiKey); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (rt *runtime) Fuzz(ctx context.Context) error {
	defer func() {
		if err := rt.client.CloseSend(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	log.Printf("[DBG] 🡱  initial msg...")
	if err := rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_Fuzz_{
				Fuzz: &fm.Clt_Msg_Fuzz{
					Resetter:  rt.models[0].GetResetter().ToProto(),
					ModelKind: fm.Clt_Msg_Fuzz_OpenAPIv3,
					Model:     rt.models[0].ToProto(),
					Usage:     os.Args,
					Seed:      []byte{42, 42, 42},
					Intensity: rt.Ntensity,
					EIDs:      rt.EIDs,
				}}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

	rt.progress = ui.NewCli()

	for {
		srv, err := rt.client.Recv()
		if err == io.EOF {
			if err := rt.models[0].GetResetter().Terminate(ctx, nil); err != nil {
				return err
			}
			if err := rt.progress.Terminate(); err != nil {
				return err
			}
			return nil
		}
		if err != nil {
			log.Println("[ERR]", err)
			return err
		}

		log.Println(srv)
		msg := srv.GetMsg().GetMsg()
		switch msg := msg.(type) {
		case *fm.Srv_Msg_Call_:
			// rt.progress.state("🙈") 🙉 🙊 🐵
			// rt.progress.Before(ui.Call)
			if err := rt.call(ctx, msg); err != nil {
				return err
			}
		case *fm.Srv_Msg_Reset_:
			if err := rt.reset(ctx, msg); err != nil {
				return err
			}
		default:
			err := fmt.Errorf("unhandled srv msg %T: %+v", msg, msg)
			log.Println("[ERR]", err)
			return err
		}
	}
}
