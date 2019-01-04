package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

func (mnk *Monkey) castPostConditions(act *RepCallDone) {
	if act.Failure {
		log.Println("[DBG] call failed, skipping checks")
		return
	}

	// Check #1: HTTP Code
	check1 := &RepValidateProgress{Details: []string{"HTTP code"}}
	log.Println("[NFO] checking", check1.Details[0])
	endpoint := mnk.Vald.Spec.Endpoints[mnk.eid].GetJson()
	status := act.Response.Response.Status
	// TODO: handle 1,2,3,4,5,XXX
	SID, ok := endpoint.Outputs[status]
	if !ok {
		check1.Failure = true
		err := fmt.Sprintf("unexpected HTTP code '%d'", status)
		check1.Details = append(check1.Details, err)
		ColorERR.Println("[NFO]", err)
	} else {
		check1.Success = true
	}
	if err := mnk.ws.cast(check1); err != nil {
		log.Fatalln("[ERR]", err)
	}
	check1 = nil
	if !ok {
		return
	}

	// Check #2: valid JSON response
	check2 := &RepValidateProgress{Details: []string{"valid JSON response"}}
	log.Println("[NFO] checking", check2.Details[0])
	var json_data interface{}
	data := []byte(act.Response.Response.Content.Text)
	if err := json.Unmarshal(data, &json_data); err != nil {
		check2.Failure = true
		check2.Details = append(check2.Details, err.Error())
		ColorERR.Println("[ERR]", err)
	} else {
		check2.Success = true
	}
	if err := mnk.ws.cast(check2); err != nil {
		log.Fatalln("[ERR]", err)
	}
	check2 = nil
	if json_data == nil {
		return
	}

	// Check #3: response validates JSON schema
	check3 := &RepValidateProgress{Details: []string{"response validates schema"}}
	log.Println("[NFO] checking", check3.Details[0])
	if errs := mnk.Vald.Spec.Schemas.Validate(SID, json_data); len(errs) != 0 {
		check3.Failure = true
		check3.Details = append(check3.Details, errs...)
		for _, e := range errs {
			ColorERR.Println(e)
		}
	} else {
		check3.Success = true
	}
	if err := mnk.ws.cast(check3); err != nil {
		log.Fatalln("[ERR]", err)
	}

	// TODO: user-provided postconditions

	checkN := &RepCallResult{Response: enumFromGo(json_data)}
	log.Println("[DBG] checks passed")
	if err := mnk.ws.cast(checkN); err != nil {
		log.Fatalln("[ERR]", err)
	}
}

func (act *ReqDoCall) exec(mnk *Monkey) (err error) {
	mnk.eid = act.EID

	if !isHARReady() {
		newHARTransport(mnk.Name)
	}

	act.updateUserAgentHeader(mnk.Name)
	if err = act.updateURL(mnk.Cfg); err != nil {
		return
	}
	act.updateHostHeader(mnk.Cfg)
	var nxt *RepCallDone
	if nxt, err = act.makeRequest(); err != nil {
		return
	}
	// FIXME: trust upstream on this
	mnk.progress.totalR++

	if err = mnk.ws.cast(nxt); err != nil {
		log.Println("[ERR]", err)
	}
	mnk.castPostConditions(nxt)
	mnk.eid = 0
	return
}

func (act *ReqDoCall) makeRequest() (nxt *RepCallDone, err error) {
	harReq := act.GetRequest()
	nxt = &RepCallDone{}
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
	nxt.Usec = uint64(us)

	if err != nil {
		//FIXME: is there a way to describe these failures in HAR 1.2?
		e := fmt.Sprintf("%#v", err.Error())
		log.Println("[NFO] ▲", e)
		nxt.Reason = e
		nxt.Failure = true
		err = nil
		return
	}

	//FIXME maybe: append(headers, fmt.Sprintf("Host: %v", resp.Host))
	//FIXME: make sure order is preserved github.com/golang/go/issues/21853
	resp := lastHAR()
	log.Println("[NFO] ▲", resp)
	nxt.Response = resp
	nxt.Success = true
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
