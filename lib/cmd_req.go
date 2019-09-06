package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"go.starlark.net/starlark"
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
			e := err.Error()
			check1.Details = append(check1.Details, e)
			log.Println("[NFO]", err)
			mnk.progress.checkFailed([]string{e})
		} else {
			check1.Success = true
			mnk.progress.checkPassed("HTTP code checked")
		}
		if err = mnk.ws.cast(check1); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if check1.Failure {
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
			e := err.Error()
			check2.Details = append(check2.Details, e)
			log.Println("[NFO]", err)
			mnk.progress.checkFailed([]string{e})
		} else {
			check2.Success = true
			mnk.progress.checkPassed("response is valid JSON")
		}
		if err = mnk.ws.cast(check2); err != nil {
			log.Println("[ERR]", err)
			return
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
			err = errors.New(strings.Join(errs, "; "))
			check3.Failure = true
			check3.Details = append(check3.Details, errs...)
			log.Println("[NFO]", err)
			mnk.progress.checkFailed(errs)
		} else {
			check3.Success = true
			mnk.progress.checkPassed("response validates JSON Schema")
		}
		if err = mnk.ws.cast(check3); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if check3.Failure {
			return
		}
	}

	// Check #N: user-provided postconditions
	{
		checkN := &RepValidateProgress{Details: []string{"user properties"}}
		log.Println("[NFO] checking", checkN.Details[0])
		userRTLang.Thread.Print = func(_ *starlark.Thread, msg string) { mnk.progress.wrn(msg) }
		userRTLang.Globals[tState] = userRTLang.ModelState
		mnk.progress.wrn(fmt.Sprintf(">>>>>> %s: %+v", tState, userRTLang.Globals[tState]))
		response := starlark.NewDict(3)
		if err := response.SetKey(starlark.String("status_code"), starlark.MakeInt(200)); err != nil {
			panic(err)
		}
		body := starlark.NewDict(1)
		if err := body.SetKey(starlark.String("id"), starlark.MakeInt(42)); err != nil {
			panic(err)
		}
		if err := response.SetKey(starlark.String("body"), body); err != nil {
			panic(err)
		}
		request := starlark.NewDict(3)
		if err := request.SetKey(starlark.String("method"), starlark.String("GET")); err != nil {
			panic(err)
		}
		if err := request.SetKey(starlark.String("path"), starlark.String("/csgo/weapons")); err != nil {
			panic(err)
		}
		if err := request.SetKey(starlark.String("route"), starlark.String("/csgo/weapons")); err != nil {
			panic(err)
		}
		if err := response.SetKey(starlark.String("request"), request); err != nil {
			panic(err)
		}
		// args := starlark.Tuple{userRTLang.ModelState, response}
		args := starlark.Tuple{response}
		for i, trigger := range userRTLang.Triggers {
			// FIXME: make predicate / action part of check name
			var shouldBeBool starlark.Value
			shouldBeBool, err = starlark.Call(userRTLang.Thread, trigger.Predicate, args, nil)
			mnk.progress.wrn(fmt.Sprintf(">>>>>> %s: %+v", tState, userRTLang.Globals[tState]))
			if err != nil {
				checkN.Failure = true
				//TODO: split on \n.s
				checkN.Details = append(checkN.Details, err.Error())
				log.Println("[NFO]", err)
				if evalErr, ok := err.(*starlark.EvalError); ok {
					bt := evalErr.Backtrace()
					mnk.progress.checkFailed([]string{bt})
				} else {
					mnk.progress.checkFailed([]string{err.Error()})
				}
				// break
			} else {
				triggered, ok := shouldBeBool.(starlark.Bool)
				if !ok {
					panic(`FIXME: thats also a check failure`)
				}
				if triggered {
					mnk.progress.nfo(fmt.Sprintf(">>> [%d] triggered", i))
					var newModelState starlark.Value
					if newModelState, err = starlark.Call(userRTLang.Thread, trigger.Action, args, nil); err != nil {
						panic(fmt.Sprintf("FIXME: %v", err))
					}
					ColorERR.Printf(">>>### State = %+v\n", newModelState)
					// if userRTLang.ModelState, ok = newModelState.(*starlark.Dict); !ok {
					// 	panic(`FIXME: thats also a check failure`)
					// }
					checkN.Success = true
					mnk.progress.checkPassed("user prop")
				} else {
					mnk.progress.nfo(fmt.Sprintf(">>> [%d] not triggered", i))
				}
			}
			if err = mnk.ws.cast(checkN); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if checkN.Failure {
				return
			}
		}
	}

	checkN := &RepCallResult{Response: enumFromGo(jsonData)}
	log.Println("[DBG] checks passed")
	mnk.progress.checksPassed()
	if err = mnk.ws.cast(checkN); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (act *ReqDoCall) exec(mnk *Monkey) (err error) {
	mnk.progress.state("Testing...")
	mnk.eid = act.EID

	if !isHARReady() {
		newHARTransport(mnk.Name)
	}

	host := mnk.Cfg.Runtime.FinalHost
	if addHost != nil {
		host = *addHost
	}
	configured, err := url.ParseRequestURI(host)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	act.updateUserAgentHeader(mnk.Name)
	if err = act.updateURL(configured); err != nil {
		return
	}
	act.updateHostHeader(configured)
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
	req, err := harReq.Request()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if addHeaderAuthorization != nil {
		req.Header.Add("Authorization", *addHeaderAuthorization)
	}

	log.Println("[NFO] ▼", harReq)
	if err = mnk.showRequest(req); err != nil {
		log.Println("[ERR]", err)
		return
	}

	start := time.Now()
	rep, err := clientReq.Do(req)
	nxt = &RepCallDone{TsDiff: uint64(time.Since(start))}

	var e string
	if err == nil {
		resp := lastHAR()
		log.Println("[NFO] ▲", resp)
		nxt.Response = resp
		nxt.Success = true
	} else {
		//FIXME: is there a way to describe these failures in HAR 1.2?
		e = err.Error()
		log.Println("[NFO] ▲", e)
		nxt.Reason = e
		nxt.Failure = true
	}

	if err = mnk.showResponse(rep, e); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (act *ReqDoCall) updateURL(configured *url.URL) (err error) {
	URL, err := url.Parse(act.Request.URL)
	if err != nil {
		log.Println("[ERR] malformed URLs are unexpected:", err)
		return
	}

	URL.Scheme = configured.Scheme
	URL.Host = configured.Host
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

func (act *ReqDoCall) updateHostHeader(configured *url.URL) {
	for i := range act.Request.Headers {
		if act.Request.Headers[i].Name == "Host" {
			act.Request.Headers[i].Value = configured.Host
			break
		}
	}
}
