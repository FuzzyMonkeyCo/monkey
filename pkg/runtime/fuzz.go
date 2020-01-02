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

func (rt *Runtime) Dial(ctx context.Context, apiKey string) (
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
		"ua", rt.binTitle,
		"apiKey", apiKey,
	)

	if rt.client, err = fm.NewFuzzyMonkeyClient(conn).Do(ctx); err != nil {
		log.Println("[ERR]", err)
		return
	}
	closer = func() error { return conn.Close() }
	return
}

func (rt *Runtime) Fuzz(ctx context.Context, ntensity uint32) error {
	defer func() {
		if err := rt.client.CloseSend(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	rt.progress = ui.NewCli()
	rt.progress.MaxTestsCount(ntensity)
	ctx = context.WithValue(ctx, "UserAgent", rt.binTitle)

	log.Printf("[DBG] 🡱  initial msg...")
	if err := rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Fuzz_{
			Fuzz: &fm.Clt_Fuzz{
				Resetter:  mdl.GetResetter().ToProto(),
				ModelKind: fm.Clt_Fuzz_OpenAPIv3,
				Model:     mdl.ToProto(),
				Usage:     os.Args,
				Seed:      []byte{42, 42, 42},
				Ntensity:  ntensity,
				EIDs:      rt.eIds,
			}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

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

		switch srv.GetMsg().(type) {
		case *fm.Srv_Call_:
			log.Println("[NFO] handling fm.Srv_Call_")
			cll := srv.GetCall()
			if err = rt.call(ctx, cll); err != nil {
				break
			}
			log.Println("[NFO] done handling fm.Srv_Call_")
			if err = rt.recvFuzzProgress(); err != nil {
				break
			}
		case *fm.Srv_Reset_:
			log.Println("[NFO] handling fm.Srv_Reset_")
			if err = rt.reset(ctx); err != nil {
				break
			}
			log.Println("[NFO] done handling fm.Srv_Reset_")
			if err = rt.recvFuzzProgress(); err != nil {
				break
			}
		default:
			err = fmt.Errorf("unhandled srv msg %T: %+v", srv.GetMsg(), srv)
			log.Println("[ERR]", err)
			break
		}
	}

	log.Println("[DBG] server dialog ended, cleaning up...")
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
