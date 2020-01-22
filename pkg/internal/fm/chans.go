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
}

// don't close a channel from the receiver side
// don't close a channel if the channel has multiple concurrent senders

func NewCh(ctx context.Context) (*Ch, error) {
	ctx, cancel := context.WithCancel(ctx)
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

	//https://godoc.org/google.golang.org/grpc#ClientStream
	// It is safe to have a goroutine calling SendMsg and another goroutine
	// calling RecvMsg on the same stream at the same time, but it is not
	// safe to call RecvMsg on the same stream in different goroutines.
	ch.rcvMsg = make(chan *Srv)
	ch.rcvErr = make(chan error)
	go func() {
		defer close(ch.rcvMsg)
		defer close(ch.rcvErr)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Println("[ERR]", err)
				ch.rcvErr <- err
				return
			default:
				log.Println("[DBG] receiving...")
				msg, err := ch.clt.Recv()
				if err != nil {
					log.Println("[DBG] received", err)
					ch.rcvErr <- err
					return
				}
				log.Println("[DBG] received!")
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
		defer close(ch.sndMsg)
		defer close(ch.sndErr)
		defer func() {
			if err := ch.clt.CloseSend(); err != nil {
				log.Println("[ERR]", err)
			}
		}()
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Println("[ERR]", err)
				ch.sndErr <- err
				return
			case r, ok := <-ch.sndMsg:
				if !ok {
					return
				}
				log.Println("[DBG] sending...")
				ch.sndErr <- ch.clt.Send(r)
				log.Println("[DBG] sent!")
			}
		}
	}()

	ch.Close = func() {
		cancel()
		if err := conn.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}
	return ch, nil
}

func (ch *Ch) RcvMsg() <-chan *Srv  { return ch.rcvMsg }
func (ch *Ch) RcvErr() <-chan error { return ch.rcvErr }

// select {
// case err := <-ch.RecvErr():
// 	   err == context.Canceled ...
// case <-time.After(X):
//     nothing recv'd avec X!
// case res = <-ch.RecvMsg():
// }

func (ch *Ch) Snd(msg *Clt) <-chan error {
	ch.sndMsg <- msg
	return ch.sndErr
}

// select {
// case err := <-ch.Snd(msg):
//    err == nil
// case <-time.After(Y):
// 	  no Send err after Y?
// }
