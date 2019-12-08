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
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const grpcHost = "do.dev.fuzzymonkey.co:7077"

func (rt *runtime) Dial(ctx context.Context, ua, apiKey string) (
	closer func() error,
	err error,
) {
	log.Println("[NFO] dialing", grpcHost)
	var conn *grpc.ClientConn
	if conn, err = grpc.DialContext(ctx, grpcHost,
		grpc.WithBlock(),
		grpc.WithTimeout(4*time.Second),
		grpc.WithInsecure(),
	); err != nil {
		log.Println("[ERR]", err)
		return
	}

	ctx = metadata.AppendToOutgoingContext(ctx,
		"ua", ua,
		"apiKey", apiKey,
	)

	if rt.client, err = fm.NewFuzzyMonkeyClient(conn).Do(ctx); err != nil {
		log.Println("[ERR]", err)
		return
	}
	closer = func() error { return conn.Close() }
	return
}

func (rt *runtime) Fuzz(ctx context.Context) error {
	defer func() {
		if err := rt.client.CloseSend(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	log.Printf("[DBG] 🡱  initial msg...")
	if err := rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_Fuzz{
				Fuzz: &fm.Clt_Fuzz{
					Resetter:  mdl.GetResetter().ToProto(),
					ModelKind: fm.Clt_Fuzz_OpenAPIv3,
					Model:     mdl.ToProto(),
					Usage:     os.Args,
					Seed:      []byte{42, 42, 42},
					Intensity: rt.Ntensity,
					EIDs:      rt.eIds,
				}}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

	rt.progress = ui.NewCli()
	rt.progress.MaxTestsCount(rt.Ntensity)

	var (
		srv *fm.Srv
		err error
	)
	for {
		if srv, err = rt.client.Recv(); err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			log.Println("[ERR]", err)
			break
		}

		log.Println("[DBG] >>>", srv)
		msg := srv.GetMsg()
		switch msg.GetMsg().(type) {
		case *fm.Srv_Msg_Call:
			log.Println("[NFO] handling Srv_Msg_Call")
			cll := msg.GetCall()
			// rt.progress.Before(ui.Call)
			if err = rt.call(ctx, cll); err != nil {
				break
			}
			log.Println("[NFO] done handling Srv_Msg_Call")
			if err = rt.recvFuzzProgress(); err != nil {
				break
			}
		case *fm.Srv_Msg_Reset_:
			log.Println("[NFO] handling Srv_Msg_Reset_")
			rst := msg.GetReset_()
			if err = rt.reset(ctx, rst); err != nil {
				break
			}
			log.Println("[NFO] done handling Srv_Msg_Reset_")
			if err = rt.recvFuzzProgress(); err != nil {
				break
			}
		default:
			err = fmt.Errorf("unhandled srv msg %T: %+v", msg.GetMsg(), msg)
			log.Println("[ERR]", err)
			break
		}
	}

	log.Println("[DBG] server dialogue ended, cleaning up...")
	if err2 := mdl.GetResetter().Terminate(ctx, nil); err2 != nil {
		log.Println("[ERR]", err2)
		return err2
	}
	if err2 := rt.progress.Terminate(); err2 != nil {
		log.Println("[ERR]", err2)
		return err2
	}
	return err
}
