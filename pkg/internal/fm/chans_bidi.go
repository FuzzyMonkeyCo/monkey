package fm

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	dialTimeout = 4 * time.Second
	rcvTimeout  = 10 * time.Second
	sndTimeout  = 10 * time.Second
)

// grpcHost is not const so its value can be set with -ldflags
var grpcHost = "do.dev.fuzzymonkey.co:7077"

// ChBiDi wraps a Clt<->Srv bidirectional gRPC channel
type ChBiDi struct {
	clt   FuzzyMonkey_DoClient
	Close func()

	rcvErr chan error
	rcvMsg chan *Srv

	sndErr chan error
	sndMsg chan *Clt
}

// NewChBiDi dials server & returns a usable ChBiDi
func NewChBiDi(ctx context.Context) (*ChBiDi, error) {
	log.Println("[NFO] dialing", grpcHost)

	options := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTimeout(dialTimeout),
		grpc.WithDefaultCallOptions(
			grpc.UseCompressor(gzip.Name),
			grpc.MaxCallRecvMsgSize(10*4194304),
		),
	}
	if !strings.HasSuffix(grpcHost, ":443") {
		options = append(options, grpc.WithInsecure())
	}
	conn, err := grpc.DialContext(ctx, grpcHost, options...)
	if err != nil {
		if err == context.DeadlineExceeded {
			err = errors.New("unreachable fuzzymonkey.co server")
		}
		log.Println("[ERR]", err)
		return nil, err
	}

	cbd := &ChBiDi{}
	if cbd.clt, err = NewFuzzyMonkeyClient(conn).Do(ctx); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	cbd.rcvMsg = make(chan *Srv)
	cbd.rcvErr = make(chan error)
	go func() {
		defer log.Println("[NFO] terminated rcv-er of Srv")
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Println("[ERR]", err)
				cbd.rcvErr <- err
				return
			default:
				msg, err := cbd.clt.Recv()
				if err != nil {
					log.Printf("[DBG] received err: %v", err)
					cbd.rcvErr <- err
					return
				}
				log.Printf("[DBG] received %T", msg.GetMsg())
				cbd.rcvMsg <- msg
			}
		}
	}()

	cbd.sndMsg = make(chan *Clt)
	cbd.sndErr = make(chan error)
	go func() {
		defer log.Println("[NFO] terminated snd-er of Clt")
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Println("[ERR]", err)
				cbd.sndErr <- err
				return
			case r, ok := <-cbd.sndMsg:
				if !ok {
					log.Println("[DBG] sndMsg is closed!")
					return
				}
				log.Printf("[DBG] sending %T...", r.GetMsg())
				err := cbd.clt.Send(r)
				log.Printf("[DBG] sent! (err: %v)", err)
				if err == io.EOF {
					// This is usually the reason & helps provide a better message
					err = context.DeadlineExceeded
				}
				cbd.sndErr <- err
			}
		}
	}()

	cbd.Close = func() {
		log.Println("[NFO] Close()-ing ChBiDi...")
		cancel()
		if err := conn.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}

	return cbd, nil
}

// Receive returns a Srv message and an error
func (cbd *ChBiDi) Receive(ctx context.Context) (msg *Srv, err error) {
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}
	select {
	case err = <-cbd.rcvErr:
	case msg = <-cbd.rcvMsg:
	case <-time.After(rcvTimeout):
		err = os.ErrDeadlineExceeded
	}
	return
}

// Send sends a Clt message, returning an error
func (cbd *ChBiDi) Send(ctx context.Context, msg *Clt) (err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	cbd.sndMsg <- msg
	select {
	case <-time.After(sndTimeout):
		err = os.ErrDeadlineExceeded
	case err = <-cbd.sndErr:
	}
	return
}
