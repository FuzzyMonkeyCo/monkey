package fm

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

var grpcHost = "do.dev.fuzzymonkey.co:7077"

const grpcDialTimeout = 4 * time.Second

type Ch struct {
	clt   FuzzyMonkey_DoClient
	Close func()

	rcvErr chan error
	rcvMsg chan *Srv

	sndErr chan error
	sndMsg chan *Clt

	ctx context.Context
}

// don't close a channel from the receiver side
// don't close a channel if the channel has multiple concurrent senders

func NewCh(ctx context.Context) (*Ch, error) {
	log.Println("[NFO] dialing", grpcHost)
	conn, err := grpc.DialContext(ctx, grpcHost,
		grpc.WithBlock(),
		grpc.WithTimeout(grpcDialTimeout),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	ch := &Ch{}
	if ch.clt, err = NewFuzzyMonkeyClient(conn).Do(ctx); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	//https://godoc.org/google.golang.org/grpc#ClientStream
	// It is safe to have a goroutine calling SendMsg and another goroutine
	// calling RecvMsg on the same stream at the same time, but it is not
	// safe to call RecvMsg on the same stream in different goroutines.
	ch.rcvMsg = make(chan *Srv)
	ch.rcvErr = make(chan error)
	go func() {
		defer log.Println("[NFO] terminated rcv-er of Srv")
		// defer close(ch.rcvSrv)
		// defer close(ch.rcvErr)
		// defer cancel()
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Println("[ERR]", err)
				ch.rcvErr <- err
				return
			default:
				msg, err := ch.clt.Recv()
				if err != nil {
					log.Printf("[DBG] received err: %v", err)
					ch.rcvErr <- err
					return
				}
				log.Printf("[DBG] received %T", msg.GetMsg())
				ch.rcvMsg <- msg
			}
		}
	}()

	// It is safe to have a goroutine calling SendMsg and another goroutine
	// calling RecvMsg on the same stream at the same time, but it is not safe
	// to call SendMsg on the same stream in different goroutines. It is also
	// not safe to call CloseSend concurrently with SendMsg.
	ch.sndMsg = make(chan *Clt)
	ch.sndErr = make(chan error)
	go func() {
		defer log.Println("[NFO] terminated snd-er of Clt")
		// defer close(ch.sndMsg)
		// defer close(ch.sndErr)
		// defer func() {
		// 	if err := ch.clt.CloseSend(); err != nil {
		// 		log.Println("[ERR]", err)
		// 	}
		// }()
		// defer cancel()
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Println("[ERR]", err)
				ch.sndErr <- err
				return
			case r, ok := <-ch.sndMsg:
				if !ok {
					log.Println("[DBG] sndMsg is closed!")
					return
				}
				log.Printf("[DBG] sending %T...", r.GetMsg())
				err := ch.clt.Send(r)
				log.Printf("[DBG] sent! (err: %v)", err)
				ch.sndErr <- err
			}
		}
	}()

	ch.Close = func() {
		log.Println("[NFO] ch.Close()-ing...")
		cancel()
		if err := conn.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}
	ch.ctx = ctx
	return ch, nil
}

func (ch *Ch) RcvMsg() <-chan *Srv  { return ch.rcvMsg }
func (ch *Ch) RcvErr() <-chan error { return ch.rcvErr }

func (ch *Ch) Snd(msg *Clt) <-chan error {
	if err := ch.ctx.Err(); err != nil {
		log.Printf("[NFO] error before sending: %v", err)
	} else {
		ch.sndMsg <- msg
	}
	return ch.sndErr
}
