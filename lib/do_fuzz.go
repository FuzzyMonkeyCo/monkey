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

	wsMsgUID++
	// reqUID := wsMsgUID
	msg := &Msg{UID: wsMsgUID}

	switch req.(type) {
	case *DoFuzz:
		msg.Msg = &Msg_Fuzz{Fuzz: req.(*DoFuzz)}
	case *RepCallDone:
		msg.Msg = &Msg_CallDone{CallDone: req.(*RepCallDone)}
	case *RepCallResult:
		msg.Msg = &Msg_CallResult{CallResult: req.(*RepCallResult)}
	case *RepResetProgress:
		msg.Msg = &Msg_ResetProgress{ResetProgress: req.(*RepResetProgress)}
	case *RepValidateProgress:
		msg.Msg = &Msg_ValidateProgress{ValidateProgress: req.(*RepValidateProgress)}
	case *SUTMetrics:
		msg.Msg = &Msg_Metrics{Metrics: req.(*SUTMetrics)}
	default:
		err = fmt.Errorf("unexpected req: %#v", req)
		return
	}

	log.Println("[DBG] encoding", msg)
	payload, err := proto.Marshal(msg)
	if err != nil {
		return
	}

	log.Printf("[DBG] 🡱 sending %dB\n", len(payload))
	ws.req <- payload
	return
}

func (act *DoFuzz) exec(mnk *Monkey) (err error) {
	act.Cfg = mnk.Cfg
	act.Spec = mnk.Vald.Spec
	if err = ws.cast(act); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func FuzzNext(mnk *Monkey, curr Action) (nxt Action, err error) {
	// Sometimes sets mnk.cfg.Runtime.Final* fields
	log.Printf(">>> curr %#v\n", curr)
	if err = curr.exec(mnk); err != nil {
		ws.err2 <- err
		return
	}

rcv:
	start := time.Now()
	select {
	case payload := <-ws.rep:
		log.Println("[DBG] 🡳", time.Since(start), payload[:4])
		msg := &Msg{}
		if err = proto.Unmarshal(payload, msg); err != nil {
			log.Println("[ERR]", err)
			return
		}

		// if msg.GetUID() != reqUID {
		// 	err = errors.New("bad dialog sequence number")
		// 	return
		// }

		switch msg.GetMsg().(type) {
		case *Msg_DoReset:
			nxt = msg.GetDoReset()
		case *Msg_DoCall:
			nxt = msg.GetDoCall()
		case *Msg_Err400:
			err = errors.New("400 Bad Request")
		case *Msg_Err401:
			err = errors.New("401 Unauthorized")
		case *Msg_Err403:
			err = errors.New("403 Forbidden")
		case *Msg_Err500:
			err = errors.New("500 Internal Server Error")
		case *Msg_FuzzProgress:
			nxt = msg.GetFuzzProgress()
			if err = nxt.exec(mnk); err != nil {
				return
			}
			if r := nxt.(*FuzzProgress); r.GetFailure() || r.GetSuccess() {
				return
			}
			nxt = nil
			goto rcv
		default:
			err = fmt.Errorf("unexpected msg: %#v", msg)
		}
		if err != nil {
			log.Println("[ERR]", err)
			return
		}

	case err = <-ws.err:
		log.Println("[ERR] 🡳", time.Since(start), err)
		return
	case <-time.After(15 * time.Second):
		err = errors.New("ws call timeout")
		log.Println("[ERR]", err)
		return
	}

	return FuzzNext(mnk, nxt)
}

func (act *FuzzProgress) exec(mnk *Monkey) (err error) {
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
			log.Printf("[DBG] recv %dB: %s...\n", len(rep), rep[:4])
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
					log.Println("[DBG] <-req", data[:4])
					start := time.Now()
					if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
						log.Println("[ERR]", err)
						ws.err <- err
						return
					}
					log.Println("[DBG] sent", len(data), "in", time.Since(start))
				case <-ws.ping:
					log.Println("[DBG] <-ping")
					data := []byte(`PONG`)
					if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Println("[ERR]", err)
						ws.err <- err
						return
					}
					once.Do(func() { pong <- struct{}{} })
				case err := <-ws.err2:
					log.Println("[DBG] <-err2")
					data := []byte(err.Error())
					if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Println("[ERR]", err)
						return
					}
				case <-ws.done:
					log.Println("[DBG] <-done")
					return
				case <-time.After(srvTimeout):
					ColorERR.Println("API server took too long to respond.")
					log.Fatalln("[ERR] srvTimeout")
					return
				}
			}
		}()
		log.Println("[DBG] ws manager ending")
		if err := c.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	select {
	case <-pong:
		log.Println("[DBG] <-pong!")
		close(pong)
		return nil
	case err := <-ws.err:
		log.Println("[DBG] <-err!")
		return err
	case <-time.After(srvTimeout):
		err := errors.New("timeout waiting for PING from server")
		log.Println("[ERR]", err)
		return err
	}
}
