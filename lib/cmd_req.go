package lib

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

func (act *RepCallDone) exec(mnk *Monkey) (nxt Action, err error) {
	return
}

func (act *ReqDoCall) exec(mnk *Monkey) (nxt Action, err error) {
	if !isHARReady() {
		newHARTransport(mnk.Name)
	}

	act.updateUserAgentHeader(mnk.Name)
	if err = act.updateURL(mnk.Cfg); err != nil {
		return
	}
	act.updateHostHeader(mnk.Cfg)
	if nxt, err = act.makeRequest(); err != nil {
		return
	}
	totalR++
	return
}

func (act *ReqDoCall) makeRequest() (nxt *RepCallDone, err error) {
	harReq := act.GetRequest()
	nxt = &RepCallDone{Usec: 42, Response: &HAR_Entry{}, Failure: true}
	r, err := harReq.Request()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] ▼", harReq)
	start := time.Now()
	_, err = clientReq.Do(r)
	us := time.Since(start)
	log.Println("[NFO] ❙", us)
	nxt = &RepCallDone{Usec: uint64(us)}

	if err != nil {
		//FIXME: is there a way to describe these failures in HAR 1.2?
		e := fmt.Sprintf("%#v", err.Error())
		log.Println("[NFO] ▲", e)
		nxt.Reason = e
		err = nil
		return
	}

	//FIXME maybe: append(headers, fmt.Sprintf("Host: %v", resp.Host))
	//FIXME: make sure order is preserved github.com/golang/go/issues/21853
	resp := lastHAR()
	log.Println("[NFO] ▲", resp)
	nxt.Response = resp
	return
}

func (act *ReqDoCall) updateURL(cfg *UserCfg) (err error) {
	URL, err := url.Parse(act.Request.URL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	// TODO: if host is an IPv6 then it has to be braced with []
	URL.Host = cfg.Runtime.FinalHost + ":" + cfg.Runtime.FinalPort
	act.Request.URL = URL.String()
	return
}

func (act *ReqDoCall) updateUserAgentHeader(ua string) {
	for i := range act.Request.Headers {
		if act.Request.Headers[i].Name == "User-Agent" {
			if strings.HasPrefix(act.Request.Headers[i].Value, "FuzzyMonkey.co/") {
				act.Request.Headers[i].Value = ua
				break
			}
		}
	}
}

func (act *ReqDoCall) updateHostHeader(cfg *UserCfg) {
	for i := range act.Request.Headers {
		if act.Request.Headers[i].Name == "Host" {
			act.Request.Headers[i].Value = cfg.Runtime.FinalHost + ":" + cfg.Runtime.FinalPort
			break
		}
	}
}
