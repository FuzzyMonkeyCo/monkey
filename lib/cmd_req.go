package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
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
		log.Printf("[NFO] checking %d user properties", len(userRTLang.Triggers))
		var response starlark.Value
		if response, err = slValueFromHAR(act.Response); err != nil {
			log.Println("[ERR]", err)
			return
		}
		args := starlark.Tuple{userRTLang.ModelState, response}
		userRTLang.Thread.Print = func(_ *starlark.Thread, msg string) { mnk.progress.wrn(msg) }
		for i, trigger := range userRTLang.Triggers {
			checkN := &RepValidateProgress{Details: []string{fmt.Sprintf("user property #%d: %q", i, trigger.Name.GoString())}}
			log.Println("[NFO] checking", checkN.Details[0])
			var shouldBeBool starlark.Value
			if shouldBeBool, err = starlark.Call(userRTLang.Thread, trigger.Predicate, args, nil); err == nil {
				if triggered, ok := shouldBeBool.(starlark.Bool); ok {
					if triggered {
						var newModelState starlark.Value
						if newModelState, err = starlark.Call(userRTLang.Thread, trigger.Action, args, nil); err == nil {
							switch newModelState := newModelState.(type) {
							case starlark.NoneType:
								checkN.Success = true
								mnk.progress.checkPassed(checkN.Details[0])
							case *modelState:
								userRTLang.ModelState = newModelState
								checkN.Success = true
								mnk.progress.checkPassed(checkN.Details[0])
							default:
								checkN.Failure = true
								err = fmt.Errorf("expected action %q (of %s) to return a ModelState, got: %v",
									trigger.Action.Name(), checkN.Details[0], newModelState)
								e := err.Error()
								checkN.Details = append(checkN.Details, e)
								log.Println("[NFO]", err)
								mnk.progress.checkFailed([]string{e})
							}
						} else {
							checkN.Failure = true
							//TODO: split on \n.s or you know create a type better than []string
							if evalErr, ok := err.(*starlark.EvalError); ok {
								checkN.Details = append(checkN.Details, evalErr.Backtrace())
							} else {
								checkN.Details = append(checkN.Details, err.Error())
							}
							log.Println("[NFO]", err)
							mnk.progress.checkFailed(checkN.Details[1:])
						}
					} else {
						mnk.progress.checkSkipped(checkN.Details[0])
					}
				} else {
					checkN.Failure = true
					err = fmt.Errorf("expected predicate to return a Bool, got: %v", shouldBeBool)
					e := err.Error()
					checkN.Details = append(checkN.Details, e)
					log.Println("[NFO]", err)
					mnk.progress.checkFailed([]string{e})
				}
			} else {
				checkN.Failure = true
				//TODO: split on \n.s or you know create a type better than []string
				if evalErr, ok := err.(*starlark.EvalError); ok {
					checkN.Details = append(checkN.Details, evalErr.Backtrace())
				} else {
					checkN.Details = append(checkN.Details, err.Error())
				}
				log.Println("[NFO]", err)
				mnk.progress.checkFailed(checkN.Details[1:])
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

	// Check #Z: all checks passed
	checkZ := &RepCallResult{Response: enumFromGo(jsonData)}
	if err = mnk.ws.cast(checkZ); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Println("[DBG] checks passed")
	mnk.progress.checksPassed()
	return
}

// CallCapturer is not CastCapturer {Request(), ..Wait?}
type CallCapturer interface {
	Request() starlark.Value
	Response() starlark.Value
}

type CallCaptureShower interface {
	ShowRequest(func(string, ...interface{})) error
	ShowResponse(func(string, ...interface{})) error
}

type transportCapturer struct {
	req, rep starlark.Value
	err      string
	elapsed  time.Duration

	showf func(string, ...interface{})
}

func newHTTPTCap(showf func(string, ...interface{})) *transportCapturer {
	return &transportCapturer{
		req:   starlark.None,
		showf: showf,
	}
}

// Request/Response somewhat follow python's `requests` API
var _ CallCapturer = (*transportCapturer)(nil)

func (c *transportCapturer) Request() starlark.Value  { return c.req }
func (c *transportCapturer) Response() starlark.Value { return c.rep }

var _ CallCaptureShower = (*transportCapturer)(nil)

func (c *transportCapturer) ShowRequest(showf func(string, ...interface{})) error {
	// TODO: output `curl` requests when showing counterexample
	//   https://github.com/sethgrid/gencurl
	//   https://github.com/moul/http2curl
	dump, err := httputil.DumpRequestOut(r, false)
	if err != nil {
		// MUST log in caller
		return err
	}
	// show(string(dump))
	showf("%s", dump)
	return nil
}

func (c *transportCapturer) ShowResponse(showf func(string, ...interface{})) error {
	if r == nil {
		panic("FIXME")
	}

	dump, err := httputil.DumpResponse(r, false)
	if err != nil {
		// MUST log in caller
		return err
	}
	// show(string(dump))
	showf("%s", dump)
	return nil
}

func (c *transportCapturer) request(r *http.Request) error {
	return nil
}

func (c *transportCapturer) response(r *http.Response) error {
	return nil
}

var _ http.RoundTripper = (*transportCapturer)(nil)

func (c *transportCapturer) RoundTrip(req *http.Request) (rep *http.Response, err error) {
	if err = c.request(req); err != nil {
		return
	}

	start := time.Now()
	if rep, err = http.DefaultTransport.RoundTrip(req); err != nil {
		return
	}
	elapsed := time.Since(start)
	c.elapsed = elapsed

	err = c.response(rep)
	return
}

func (act *ReqDoCall) exec(mnk *Monkey) (err error) {
	mnk.progress.state("🙈")
	mnk.eid = act.EID

	tcap := newHTTPTCap(mnk.progress.showf)

	var nxt *RepCallDone
	if nxt, err = tcap.makeRequest(act.GetRequest(), mnk.Cfg.Runtime.FinalHost); err != nil {
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

func (c *transportCapturer) makeRequest(harReq *HAR_Request, host string) (nxt *RepCallDone, err error) {
	req, err := harReq.Request()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if addHeaderAuthorization != nil {
		req.Header.Add("Authorization", *addHeaderAuthorization)
	}

	if addHost != nil {
		host = *addHost
	}
	configured, err := url.ParseRequestURI(host)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	updateUserAgentHeader(req)
	updateHostHeader(req, configured)
	if err = updateURL(req, configured); err != nil {
		return
	}

	log.Println("[NFO] ▼", harReq)
	if err = c.ShowRequest(c.showf); err != nil {
		log.Println("[ERR]", err)
		return
	}

	_, err = (&http.Client{
		Transport: c,
	}).Do(req)

	nxt = &RepCallDone{TsDiff: uint64(c.elapsed)}
	if err == nil {
		resp := c.Response()
		log.Println("[NFO] ▲", resp)
		// FIXME: nxt.Response = resp
		nxt.Success = true
	} else {
		c.err = err.Error()
		log.Println("[NFO] ▲", c.err)
		nxt.Reason = c.err
		nxt.Failure = true
	}

	if err = c.ShowResponse(c.showf); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func updateURL(r *http.Request, configured *url.URL) (err error) {
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

func updateUserAgentHeader(r *http.Request) {
	for i := range act.Request.Headers {
		if act.Request.Headers[i].Name == "User-Agent" {
			if strings.HasPrefix(act.Request.Headers[i].Value, "FuzzyMonkey.co/") {
				act.Request.Headers[i].Value = binTitle
				break
			}
		}
	}
}

func updateHostHeader(r *http.Request, configured *url.URL) {
	for i := range act.Request.Headers {
		if act.Request.Headers[i].Name == "Host" {
			act.Request.Headers[i].Value = configured.Host
			break
		}
	}
}

func slValueFromHAR(entry *HAR_Entry) (r starlark.Value, err error) {
	// FIXME: skip HAR entirely
	req := starlark.NewDict(4)
	method := starlark.String(entry.GetRequest().GetMethod())
	if err = req.SetKey(starlark.String("method"), method); err != nil {
		return
	}
	url := starlark.String(entry.GetRequest().GetURL())
	if err = req.SetKey(starlark.String("url"), url); err != nil {
		return
	}
	reqHs := entry.GetRequest().GetHeaders()
	reqHeaders := starlark.NewDict(len(reqHs))
	for _, h := range reqHs {
		k, v := starlark.String(h.GetName()), starlark.String(h.GetValue())
		// FIXME: turn v into []v, once HAR is out of the way
		//     ContentLength int64
		//     TransferEncoding []string
		//     Host string
		if err = reqHeaders.SetKey(k, v); err != nil {
			return
		}
	}
	if err = req.SetKey(starlark.String("headers"), reqHeaders); err != nil {
		return
	}
	// req['content'] as bytes?
	if txt := entry.GetRequest().GetPostData().GetText(); txt != "" {
		var reqJSON interface{}
		if err = json.Unmarshal([]byte(txt), &reqJSON); err != nil {
			return
		}
		var valJSON starlark.Value
		if valJSON, err = slValueFromInterface(reqJSON); err != nil {
			return
		}
		if err = req.SetKey(starlark.String("json"), valJSON); err != nil {
			return
		}
	} else {
		if err = req.SetKey(starlark.String("json"), starlark.None); err != nil {
			return
		}
	}
	// TODO:
	//     // Response is the redirect response which caused this request
	//     // to be created. This field is only populated during client
	//     // redirects.
	//     Response *Response
	//     // contains filtered or unexported fields

	rep := starlark.NewDict(5)
	statusCode := starlark.MakeUint(uint(entry.GetResponse().GetStatus()))
	if err = rep.SetKey(starlark.String("status_code"), statusCode); err != nil {
		return
	}
	reason := starlark.String(entry.GetResponse().GetStatusText())
	if err = rep.SetKey(starlark.String("reason"), reason); err != nil {
		return
	}
	repHs := entry.GetResponse().GetHeaders()
	repHeaders := starlark.NewDict(len(repHs))
	for _, h := range repHs {
		k, v := starlark.String(h.GetName()), starlark.String(h.GetValue())
		// FIXME: turn v into []v, once HAR is out of the way
		//   ContentLength int64
		//   TransferEncoding []string
		if err = repHeaders.SetKey(k, v); err != nil {
			return
		}
	}
	if err = rep.SetKey(starlark.String("headers"), repHeaders); err != nil {
		return
	}
	// rep['content'] as bytes?
	if txt := entry.GetResponse().GetContent().GetText(); txt != "" {
		var repJSON interface{}
		if err = json.Unmarshal([]byte(txt), &repJSON); err != nil {
			return
		}
		var valJSON starlark.Value
		if valJSON, err = slValueFromInterface(repJSON); err != nil {
			return
		}
		if err = rep.SetKey(starlark.String("json"), valJSON); err != nil {
			return
		}
	} else {
		if err = rep.SetKey(starlark.String("json"), starlark.None); err != nil {
			return
		}
	}
	// rep['history'][]Rep (redirects)?
	if err = rep.SetKey(starlark.String("request"), req); err != nil {
		return
	}
	//     // TLS contains information about the TLS connection on which the
	//     // response was received. It is nil for unencrypted responses.
	//     // The pointer is shared between responses and should not be
	//     // modified.
	//     TLS *tls.ConnectionState
	// }

	r = rep
	return
}
