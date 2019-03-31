package lib

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
)

func (ws *wsState) cast(req Action) (err error) {
	msg := &Msg{Ts: NowMonoNano()}
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
	var payload []byte
	if payload, err = proto.Marshal(msg); err != nil {
		// Log err in caller
		return
	}

	log.Printf("[DBG] 🡱 sending %dB\n", len(payload))
	ws.req <- payload
	return
}

func (mnk *Monkey) FuzzingLoop(act Action) (done *FuzzProgress, err error) {
	for {
		// Sometimes sets mnk.cfg.Runtime.Final* fields
		log.Printf("[DBG] >>> act %#v\n", act)
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
	log.Println("[DBG] >>> FuzzProgress", act)
	currentLane := lane{
		t: act.GetTotalTestsCount(),
		r: act.GetTestCallsCount(),
		c: act.GetCallChecksCount(),
	}

	var str string

	if act.GetLastCallSuccess() {
		str = "✓"
	}
	if act.GetLastCallFailure() {
		str = "✗"
	}

	switch {
	case mnk.progress.shrinkingFrom == nil && act.GetShrinking():
		mnk.progress.shrinkingFrom = &mnk.progress.lastLane
		str += "\n"
	case mnk.progress.lastLane.t != currentLane.t:
		str += " "
	}

	mnk.progress.lastLane = currentLane
	if act.GetFailure() || act.GetSuccess() {
		mnk.progress.totalR = act.GetTotalCallsCount()
		mnk.progress.totalC = act.GetTotalChecksCount()
	}

	fmt.Print(str)
	return
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

func (mnk *Monkey) Outcome(act *FuzzProgress) (success bool) {
	p := mnk.progress
	os.Stdout.Write([]byte{'\n'})
	ColorWRN.Println(
		"Ran", p.lastLane.t, plural("test", p.lastLane.t),
		"totalling", p.totalR, plural("request", p.totalR),
		"and", p.totalC, plural("check", p.totalC),
		"in", time.Since(p.start))

	if act.GetSuccess() {
		ColorNFO.Println("No bugs found... yet.")
		success = true
		return
	}
	if !act.GetFailure() {
		panic(`there should be success!`)
	}

	var d, m uint32
	if p.shrinkingFrom == nil {
		d = p.lastLane.t
	} else {
		d = p.shrinkingFrom.t
		m = p.lastLane.t - d
	}
	ColorERR.Printf("A bug reproducible in %d HTTP %s", p.lastLane.r, plural("request", p.lastLane.r))
	ColorERR.Printf(" was detected after %d %s ", d, plural("test", d))
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

	log.Println("[NFO] dialing", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}

	mnk.progress = newProgress()
	mnk.ws = newWS(u, c)

	select {
	case err := <-mnk.ws.err:
		log.Println("[DBG] <-err!")
		return err
	default:
		return nil
	}
}
