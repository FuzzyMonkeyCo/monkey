package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

func (mnk *Monkey) castPostConditions(act *RepCallDone) (err error) {
	if act.Failure {
		log.Println("[DBG] call failed, skipping checks")
		return
	}

	var SID sid
	// Check #1: HTTP Code
	{
		check1 := &RepValidateProgress{Details: []string{"HTTP code"}}
		log.Println("[NFO] checking", check1.Details[0])
		endpoint := mnk.Vald.Spec.Endpoints[mnk.eid].GetJson()
		status := act.Response.Response.Status
		// TODO: handle 1,2,3,4,5,XXX
		var ok bool
		if SID, ok = endpoint.Outputs[status]; !ok {
			check1.Failure = true
			err = fmt.Errorf("unexpected HTTP code '%d'", status)
			check1.Details = append(check1.Details, err.Error())
			ColorERR.Println("[NFO]", err)
		} else {
			check1.Success = true
		}
		if e := mnk.ws.cast(check1); e != nil {
			log.Println("[ERR]", e)
			return e
		}
		if check1.Failure || !ok {
			return
		}
	}

	var jsonData interface{}
	// Check #2: valid JSON response
	{
		check2 := &RepValidateProgress{Details: []string{"valid JSON response"}}
		log.Println("[NFO] checking", check2.Details[0])
		data := []byte(act.Response.Response.Content.Text)
		if err = json.Unmarshal(data, &jsonData); err != nil {
			check2.Failure = true
			check2.Details = append(check2.Details, err.Error())
			ColorERR.Println("[ERR]", err)
		} else {
			check2.Success = true
		}
		if e := mnk.ws.cast(check2); e != nil {
			log.Println("[ERR]", e)
			return e
		}
		if check2.Failure || jsonData == nil {
			return
		}
	}

	// Check #3: response validates JSON schema
	{
		check3 := &RepValidateProgress{Details: []string{"response validates schema"}}
		log.Println("[NFO] checking", check3.Details[0])
		if errs := mnk.Vald.Spec.Schemas.Validate(SID, jsonData); len(errs) != 0 {
			err = errors.New(errs[0])
			check3.Failure = true
			check3.Details = append(check3.Details, errs...)
			for _, e := range errs {
				ColorERR.Println(e)
			}
		} else {
			check3.Success = true
		}
		if e := mnk.ws.cast(check3); e != nil {
			log.Println("[ERR]", e)
			return e
		}
		if check3.Failure {
			return
		}
	}

	// TODO: user-provided postconditions

	checkN := &RepCallResult{Response: enumFromGo(jsonData)}
	log.Println("[DBG] checks passed")
	if err = mnk.ws.cast(checkN); err != nil {
		log.Println("[ERR]", err)
	}
	return
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
	if nxt, err = act.makeRequest(mnk); err != nil {
		return
	}

	if err = mnk.ws.cast(nxt); err != nil {
		log.Println("[ERR]", err)
		return
	}
	err = mnk.castPostConditions(nxt)
	mnk.eid = 0
	return
}

func (act *ReqDoCall) makeRequest(mnk *Monkey) (nxt *RepCallDone, err error) {
	harReq := act.GetRequest()
	nxt = &RepCallDone{}
	req, err := harReq.Request()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] ▼", harReq)
	if err = mnk.showRequest(req); err != nil {
		log.Println("[ERR]", err)
		return
	}

	start := time.Now()
	rep, err := clientReq.Do(req)
	nxt.TsDiff = uint64(time.Since(start))

	var e string
	if err != nil {
		//FIXME: is there a way to describe these failures in HAR 1.2?
		e = err.Error()
		log.Println("[NFO] ▲", e)
		nxt.Reason = e
		nxt.Failure = true
	}

	//FIXME maybe: append(headers, fmt.Sprintf("Host: %v", resp.Host))
	//FIXME: make sure order is preserved github.com/golang/go/issues/21853
	var resp *HAR_Entry
	if err == nil {
		resp = lastHAR()
		log.Println("[NFO] ▲", resp)
	}

	if err = mnk.showResponse(rep, e); err != nil {
		log.Println("[ERR]", err)
		return
	}

	if err != nil {
		return nxt, nil
	}
	nxt.Response = resp
	nxt.Success = true
	return
}

func (act *ReqDoCall) updateURL(cfg *UserCfg) (err error) {
	URL, err := url.Parse(act.Request.URL)
	if err != nil {
		log.Println("[ERR]", err)
		// Malformed URLs are unexpected
		panic(err)
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
