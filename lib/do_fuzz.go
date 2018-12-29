package lib

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	headerUserAgent = "User-Agent"
	headerXAPIKey   = "X-Api-Key"
)

var ws *wsState
var wsMsgUID uint32

type wsState struct {
	req  chan []byte
	rep  chan []byte
	err  chan error
	err2 chan error
	ping chan struct{}
	done chan struct{}
}

func (ws *wsState) cast(req Action) (err error) {
	// NOTE: log in caller
	// FIXME: use buffers?

	msg := &Msg{UID: wsMsgUID}

	switch req.(type) {
	case *RepCallResult:
		msg.Msg = &Msg_CallResult{CallResult: req.(*RepCallResult)}
	case *RepValidateProgress:
		msg.Msg = &Msg_ValidateProgress{ValidateProgress: req.(*RepValidateProgress)}
	default:
		err = fmt.Errorf("unexpected req: %#v", req)
	}
	if err != nil {
		return
	}

	log.Println(">>> about to CAST", msg)
	payload, err := proto.Marshal(msg)
	if err != nil {
		return
	}

	log.Printf("[DBG] 🡱 sending %dB\n", len(payload))
	ws.req <- payload
	return
}

func (ws *wsState) call(req Action, mnk *Monkey) (rep Action, err error) {
	// NOTE: log in caller
	// FIXME: use buffers?

	wsMsgUID++
	// reqUID := wsMsgUID
	msg := &Msg{UID: wsMsgUID}

	switch req.(type) {
	case *DoFuzz:
		msg.Msg = &Msg_Fuzz{Fuzz: req.(*DoFuzz)}
	case *RepResetProgress:
		msg.Msg = &Msg_ResetProgress{ResetProgress: req.(*RepResetProgress)}
	case *RepCallDone:
		msg.Msg = &Msg_CallDone{CallDone: req.(*RepCallDone)}
	default:
		err = fmt.Errorf("unexpected req: %#v", req)
	}
	if err != nil {
		return
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		return
	}

	log.Printf("[DBG] 🡱 sending %dB\n", len(payload))
	ws.req <- payload
rcv:
	start := time.Now()
	select {
	case payload = <-ws.rep:
		log.Println("[DBG] 🡳", time.Since(start), payload[:4])
		if err = proto.Unmarshal(payload, msg); err != nil {
			return
		}

		// if msg.GetUID() != reqUID {
		// 	err = errors.New("bad dialog sequence number")
		// 	return
		// }

		switch msg.GetMsg().(type) {
		case *Msg_Err500:
			err = errors.New("500 Internal Server Error")
		case *Msg_Err400:
			err = errors.New("400 Bad Request")
		case *Msg_Err401:
			err = errors.New("401 Unauthorized")
		case *Msg_Err403:
			err = errors.New("403 Forbidden")
		case *Msg_DoReset:
			rep = msg.GetDoReset()
		case *Msg_DoCall:
			rep = msg.GetDoCall()
		case *Msg_FuzzProgress:
			rep = msg.GetFuzzProgress()
			rep.exec(mnk)
			if r := rep.(*FuzzProgress); !(r.GetFailure() || r.GetSuccess()) {
				goto rcv
			}
		default:
			err = fmt.Errorf("unexpected msg: %#v", msg)
		}

	case err = <-ws.err:
		log.Println("[DBG] 🡳", time.Since(start), err)
	case <-time.After(15 * time.Second):
		err = errors.New("ws call timeout")
	}
	return
}

func (act *DoFuzz) exec(mnk *Monkey) (nxt Action, err error) {
	act.Cfg = mnk.Cfg
	act.Spec = mnk.Vald.Spec
	nxt = act
	return
}

func (act *RepResetProgress) exec(mnk *Monkey) (nxt Action, err error) {
	return
}

func FuzzNext(mnk *Monkey, curr Action) (nxt Action, err error) {
	// Sometimes sets mnk.cfg.Runtime.Final* fields
	log.Printf(">>> curr %#v\n", curr)
	if nxt, err = curr.exec(mnk); err != nil {
		ws.err2 <- err
		return
	}
	log.Printf(">>> nxt %#v\n", nxt)

	// FIXME: find a better place for this call
	if called, ok := nxt.(*RepCallDone); ok {
		called.castPostConditions(mnk)
		mnk.EID = 0
	}

	if nxt == nil {
		return
	}

	if nxt, err = ws.call(nxt, mnk); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (act *FuzzProgress) exec(mnk *Monkey) (nxt Action, err error) {
	log.Println(">>> FuzzProgress", act)
	lastLane = lane{T: act.TotalTestsCount, R: act.TestCallsCount}
	return
}

func (act *FuzzProgress) Outcome() int {
	os.Stdout.Write([]byte{'\n'})
	fmt.Printf("Ran %d tests totalling %d requests\n", lastLane.T, totalR)

	if act.GetFailure() {
		d, m := shrinkingFrom.T, lastLane.T-shrinkingFrom.T
		if m != 1 {
			fmt.Printf("A bug was detected after %d tests then shrunk %d times!\n", d, m)
		} else {
			fmt.Printf("A bug was detected after %d tests then shrunk once!\n", d)
		}
		return 6
	}

	if !act.GetSuccess() {
		log.Fatalln("[ERR] there should be success!")
	}
	fmt.Println("No bugs found... yet.")
	return 0
}

func NewWS(cfg *UserCfg, URL, ua string) error {
	u, err := url.Parse(URL)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}
	headers := http.Header{
		headerUserAgent: {ua},
		headerXAPIKey:   {cfg.ApiKey},
	}

	log.Println("[NFO] connecting to", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}

	ws = &wsState{
		req:  make(chan []byte, 1),
		rep:  make(chan []byte, 1),
		err:  make(chan error, 1),
		err2: make(chan error, 1),
		ping: make(chan struct{}, 1),
		done: make(chan struct{}, 1),
	}

	go func() {
		defer func() { ws.done <- struct{}{} }()
		for {
			_, rep, err := c.ReadMessage()
			if err != nil {
				log.Println("[ERR]", err)
				ws.err <- err
				return
			}
			log.Printf("recv %dB: %s...\n", len(rep), rep[:4])
			if len(rep) == 4 && string(rep) == `PING` {
				ws.ping <- struct{}{}
			} else {
				ws.rep <- rep
			}
		}
	}()

	var once sync.Once
	pong := make(chan struct{}, 1)
	srvTimeout := 15 * time.Second

	go func() {
		defer close(ws.req)
		defer close(ws.rep)
		defer close(ws.err)
		defer close(ws.err2)
		defer close(ws.ping)
		defer close(ws.done)

		func() {
			for {
				select {
				case data := <-ws.req:
					log.Println("ws.req")
					if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
						log.Println("[ERR]", err)
						ws.err <- err
						return
					}
				case <-ws.ping:
					log.Println("ws.ping")
					data := []byte(`PONG`)
					if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Println("[ERR]", err)
						ws.err <- err
						return
					}
					once.Do(func() { pong <- struct{}{} })
				case err := <-ws.err2:
					log.Println("ws.err2")
					data := []byte(err.Error())
					if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Println("[ERR]", err)
						return
					}
				case <-ws.done:
					log.Println("ws.done")
					return
				case <-time.After(srvTimeout):
					log.Fatalln("[ERR] srvTimeout")
					return
				}
			}
		}()
		log.Println("ws manager ending")
		if err := c.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	select {
	case <-pong:
		log.Println("pong!")
		close(pong)
		return nil
	case err := <-ws.err:
		log.Println("err!")
		return err
	case <-time.After(srvTimeout):
		err := errors.New("timeout waiting for PING from server")
		log.Println("[ERR]", err)
		return err
	}
}
