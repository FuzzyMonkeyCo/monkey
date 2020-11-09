package fm

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
)

// grpcHost is a var so its value may be set with -ldflags' -X
var grpcHost = "do.dev.fuzzymonkey.co:7077"

// ChBiDi wraps a Clt<->Srv bidirectional gRPC channel
type ChBiDi struct {
	clt   FuzzyMonkey_DoClient
	Close func()

	rcvErr chan error
	rcvMsg chan *Srv

	sndErr chan error
	sndMsg chan *Clt

	ctx context.Context
}

// NewChBiDi dials server & returns a usable ChBiDi
func NewChBiDi(ctx context.Context) (*ChBiDi, error) {
	log.Println("[NFO] dialing", grpcHost)

	options := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTimeout(4 * time.Second), // Only for dialing
	}
	if !strings.HasSuffix(grpcHost, ":443") {
		options = append(options, grpc.WithInsecure())
	}
	conn, err := grpc.DialContext(ctx, grpcHost, options...)
	if err != nil {
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
	cbd.ctx = ctx
	return cbd, nil
}

// RcvMsg returns a Srv message channel
func (cbd *ChBiDi) RcvMsg() <-chan *Srv { return cbd.rcvMsg }

// RcvErr returns an error channel
func (cbd *ChBiDi) RcvErr() <-chan error { return cbd.rcvErr }

// Snd sends a Clt message, returning an error channel
func (cbd *ChBiDi) Snd(msg *Clt) <-chan error {
	if err := cbd.ctx.Err(); err != nil {
		log.Printf("[NFO] error before sending: %v", err)
	} else {
		cbd.sndMsg <- msg
	}
	return cbd.sndErr
}
