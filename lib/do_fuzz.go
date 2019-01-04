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
	// If no message arrive in `srvTimeout` then fail
	srvTimeout = 15 * time.Second
)

type wsState struct {
	URL    *url.URL
	c      *websocket.Conn
	msgUID uint32
	req    chan []byte
	rep    chan []byte
	err    chan error
	err2   chan error
	ping   chan struct{}
	done   chan struct{}
}

func newWS(URL *url.URL, c *websocket.Conn) *wsState {
	return &wsState{
		URL:  URL,
		c:    c,
		req:  make(chan []byte, 1),
		rep:  make(chan []byte, 1),
		err:  make(chan error, 1),
		err2: make(chan error, 1),
		ping: make(chan struct{}, 1),
		done: make(chan struct{}, 1),
	}
}

func (ws *wsState) cast(req Action) (err error) {
	ws.msgUID++
	// reqUID := wsMsgUID
	msg := &Msg{UID: ws.msgUID}

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

func (mnk *Monkey) FuzzingLoop(act Action) (done *FuzzProgress, err error) {
	for {
		// Sometimes sets mnk.cfg.Runtime.Final* fields
		log.Printf(">>> act %#v\n", act)
		if err = act.exec(mnk); err != nil {
			mnk.ws.err2 <- err
			return
		}

	rcv:
		start := time.Now()
		select {
		case payload := <-mnk.ws.rep:
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
				act = msg.GetDoReset()
			case *Msg_DoCall:
				act = msg.GetDoCall()
			case *Msg_Err400:
				err = errors.New("400 Bad Request")
			case *Msg_Err401:
				err = errors.New("401 Unauthorized")
			case *Msg_Err403:
				err = errors.New("403 Forbidden")
			case *Msg_Err500:
				err = errors.New("500 Internal Server Error")
			case *Msg_FuzzProgress:
				done = msg.GetFuzzProgress()
				if err = done.exec(mnk); err != nil {
					return
				}
				if done.GetFailure() || done.GetSuccess() {
					return
				}
				done = nil
				goto rcv
			default:
				err = fmt.Errorf("unexpected msg: %#v", msg)
			}
			if err != nil {
				log.Println("[ERR]", err)
				return
			}

		case err = <-mnk.ws.err:
			log.Println("[ERR] 🡳", time.Since(start), err)
			return
		case <-time.After(15 * time.Second):
			err = errors.New("ws call timeout")
			log.Println("[ERR]", err)
			return
		}
	}
}

func (act *DoFuzz) exec(mnk *Monkey) (err error) {
	act.Cfg = mnk.Cfg
	act.Spec = mnk.Vald.Spec
	if err = mnk.ws.cast(act); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (act *FuzzProgress) exec(mnk *Monkey) (err error) {
	log.Println(">>> FuzzProgress", act)
	mnk.progress.lastLane = lane{
		t: act.GetTotalTestsCount(),
		r: act.GetTestCallsCount(),
		c: act.GetCallChecksCount(),
	}
	if act.GetFailure() || act.GetSuccess() {
		mnk.progress.totalR = act.GetTotalCallsCount()
		mnk.progress.totalC = act.GetTotalChecksCount()
	}
	return
}

func (mnk *Monkey) Outcome(act *FuzzProgress) (success bool) {
	p := mnk.progress
	os.Stdout.Write([]byte{'\n'})
	ColorWRN.Println(
		"Ran", p.lastLane.t, "tests",
		"totalling", p.totalR, "requests",
		"and", p.totalC, "checks",
		"in", time.Since(p.start))

	if act.GetSuccess() {
		ColorNFO.Println("No bugs found... yet.")
		success = true
		return
	}
	if !act.GetFailure() {
		panic(`there should be success!`)
	}

	d := p.shrinkingFrom.t
	m := p.lastLane.t - d
	ColorERR.Printf("A bug reproducible in %d HTTP requests", p.lastLane.r)
	ColorERR.Printf(" was detected after %d tests ", d)
	switch m {
	case 0:
		ColorERR.Println("and not yet shrunk.")
	case 1:
		ColorERR.Println("then shrunk", "once.")
	default:
		ColorERR.Println("then shrunk", m, "times.")
	}
	return
}

func (mnk *Monkey) Dial(URL string) error {
	u, err := url.Parse(URL)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}
	headers := http.Header{
		"User-Agent": {mnk.Name},
		"X-Api-Key":  {mnk.Cfg.ApiKey},
	}

	log.Println("[NFO] connecting to", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}

	mnk.progress = newProgress()
	mnk.ws = newWS(u, c)

	go mnk.ws.reader()
	pong := make(chan struct{}, 1)
	go mnk.ws.writer(pong)

	select {
	case <-pong:
		log.Println("[DBG] <-pong!")
		close(pong)
		return nil
	case err := <-mnk.ws.err:
		log.Println("[DBG] <-err!")
		return err
	case <-time.After(srvTimeout):
		err := errors.New("timeout waiting for PING from server")
		log.Println("[ERR]", err)
		return err
	}
}

func (ws *wsState) reader() {
	defer func() { ws.done <- struct{}{} }()

	for {
		_, rep, err := ws.c.ReadMessage()
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
}

func (ws *wsState) writer(pong chan struct{}) {
	defer close(ws.req)
	defer close(ws.rep)
	defer close(ws.err)
	defer close(ws.err2)
	defer close(ws.ping)
	defer close(ws.done)

	func() {
		var once sync.Once
		const binMsg = websocket.BinaryMessage
		const txtMsg = websocket.TextMessage

		for {
			select {
			case data := <-ws.req:
				log.Println("[DBG] <-req", data[:4])
				start := time.Now()
				if err := ws.c.WriteMessage(binMsg, data); err != nil {
					log.Println("[ERR]", err)
					ws.err <- err
					return
				}
				log.Println("[DBG] sent", len(data), "in", time.Since(start))
			case <-ws.ping:
				log.Println("[DBG] <-ping")
				data := []byte(`PONG`)
				if err := ws.c.WriteMessage(txtMsg, data); err != nil {
					log.Println("[ERR]", err)
					ws.err <- err
					return
				}
				once.Do(func() { pong <- struct{}{} })
			case err := <-ws.err2:
				log.Println("[DBG] <-err2")
				data := []byte(err.Error())
				if err := ws.c.WriteMessage(txtMsg, data); err != nil {
					log.Println("[ERR]", err)
					return
				}
			case <-ws.done:
				log.Println("[DBG] <-done")
				return
			case <-time.After(srvTimeout):
				ColorERR.Println(ws.URL.Hostname(), "took too long to respond")
				log.Fatalln("[ERR] srvTimeout")
				return
			}
		}
	}()

	log.Println("[DBG] ws manager ending")
	if err := ws.c.Close(); err != nil {
		log.Println("[ERR]", err)
	}
}
