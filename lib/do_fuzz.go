package lib

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
)

func (ws *wsState) cast(req Action) (err error) {
	msg := &Msg{Ts: NowMonoNano()}
	switch r := req.(type) {
	case *DoFuzz:
		msg.Msg = &Msg_Fuzz{Fuzz: r}
	case *RepCallDone:
		msg.Msg = &Msg_CallDone{CallDone: r}
	case *RepCallResult:
		msg.Msg = &Msg_CallResult{CallResult: r}
	case *RepResetProgress:
		msg.Msg = &Msg_ResetProgress{ResetProgress: r}
	case *RepValidateProgress:
		msg.Msg = &Msg_ValidateProgress{ValidateProgress: r}
	case *SUTMetrics:
		msg.Msg = &Msg_Metrics{Metrics: r}
	default:
		err = fmt.Errorf("unexpected req: %#v", req)
		return
	}

	log.Println("[DBG] encoding", msg)
	var payload []byte
	if payload, err = proto.Marshal(msg); err != nil {
		// Log err in caller
		return
	}

	log.Printf("[DBG] 🡱 sending %dB\n", len(payload))
	ws.req <- payload
	return
}

func (mnk *Monkey) FuzzingLoop(act Action) (err error) {
	defer func() {
		mnk.progress.bar.Done()
		fmt.Println()
		fmt.Println()
	}()

	for {
		// Sometimes sets mnk.cfg.Runtime.FinalHost
		log.Printf("[ERR] >>> act %#v\n", act)
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

			switch msg.GetMsg().(type) {
			case *Msg_DoReset:
				act = msg.GetDoReset()
			case *Msg_DoCall:
				act = msg.GetDoCall()
			case *Msg_Error:
				e := msg.GetError()
				culprit := e.GetCulprit().String()
				if reason := e.GetReason(); reason != "" {
					err = fmt.Errorf("%s error: %s", culprit, reason)
				} else {
					err = fmt.Errorf("%s error", culprit)
				}
			case *Msg_FuzzProgress:
				done := msg.GetFuzzProgress()
				if err = done.exec(mnk); err != nil {
					return
				}
				if done.GetFailure() || done.GetSuccess() {
					return
				}
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

func (mnk *Monkey) Dial(URL string) error {
	u, err := url.ParseRequestURI(URL)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}
	headers := http.Header{
		"User-Agent": {mnk.Name},
		"X-Api-Key":  {mnk.Cfg.ApiKey},
	}

	log.Println("[NFO] dialing", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}

	mnk.progress = newProgress(mnk.Cfg.N)
	mnk.ws = newWS(u, c)

	select {
	case err := <-mnk.ws.err:
		log.Println("[DBG] <-err!")
		return err
	default:
		return nil
	}
}
