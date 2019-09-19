package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.starlark.net/starlark"
)

var (
	headerContentLength    = http.CanonicalHeaderKey("Content-Length")
	headerHost             = http.CanonicalHeaderKey("Host")
	headerTransferEncoding = http.CanonicalHeaderKey("Transfer-Encoding")
	headerUserAgent        = http.CanonicalHeaderKey("User-Agent")
)

func (mnk *Monkey) castPostConditions(act *RepCallDone) (err error) {
	if act.Failure {
		log.Println("[DBG] call failed, skipping checks")
		return
	}

	var reqrep CallCapturer = tcap
	// FIXME: turn this into a sync.errgroup with additional tasks being
	// triggers with match-all predicates andalso pure actions
	for {
		name, lambda := reqrep.CheckFirst()
		if name == "" {
			break
		}

		check := &RepValidateProgress{Details: []string{name}}
		log.Println("[NFO] checking", check.Details[0])
		success, failure := lambda(mnk)
		switch {
		case success != "":
			check.Success = true
			mnk.progress.checkPassed(success)
		case len(failure) != 0:
			check.Details = append(check.Details, failure...)
			log.Println(append([]string{"[NFO]"}, failure...))
			mnk.progress.checkFailed(failure)
		default:
			mnk.progress.checkSkipped(check.Details[0])
		}

		if err = mnk.ws.cast(check); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if check.Failure {
			return
		}
	}

	// Check #N: user-provided postconditions
	{
		log.Printf("[NFO] checking %d user properties", len(userRTLang.Triggers))
		var response starlark.Value
		if response, err = slValueFromInterface(reqrep.Response()); err != nil {
			log.Println("[ERR]", err)
			return
		}
		userRTLang.Thread.Print = func(_ *starlark.Thread, msg string) { mnk.progress.wrn(msg) }
		for i, trigger := range userRTLang.Triggers {
			checkN := &RepValidateProgress{Details: []string{fmt.Sprintf("user property #%d: %q", i, trigger.Name.GoString())}}
			log.Println("[NFO] checking", checkN.Details[0])

			var modelState1, response1 starlark.Value
			if modelState1, err = slValueCopy(userRTLang.ModelState); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if response1, err = slValueCopy(response); err != nil {
				log.Println("[ERR]", err)
				return
			}
			args1 := starlark.Tuple{modelState1, response1}

			var shouldBeBool starlark.Value
			if shouldBeBool, err = starlark.Call(userRTLang.Thread, trigger.Predicate, args1, nil); err == nil {
				if triggered, ok := shouldBeBool.(starlark.Bool); ok {
					if triggered {

						var modelState2, response2 starlark.Value
						if modelState2, err = slValueCopy(userRTLang.ModelState); err != nil {
							log.Println("[ERR]", err)
							return
						}
						if response2, err = slValueCopy(response); err != nil {
							log.Println("[ERR]", err)
							return
						}
						args2 := starlark.Tuple{modelState2, response2}

						var newModelState starlark.Value
						if newModelState, err = starlark.Call(userRTLang.Thread, trigger.Action, args2, nil); err == nil {
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
								err = fmt.Errorf("expected action %q (of %s) to return a ModelState, got: %T %v",
									trigger.Action.Name(), checkN.Details[0], newModelState, newModelState)
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
							mnk.progress.checkFailed(checkN.Details)
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
	checkZ := &RepCallResult{} //FIXME:Response: enumFromGo(jsonData)}
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
	Request() map[string]interface{}
	Response() map[string]interface{}

	// FIXME: really not sure that this belongs here:
	CheckFirst() (string, CheckerFunc)
}

// CheckerFunc TODO
type CheckerFunc func(*Monkey) (string, []string)

// CallCaptureShower TODO
type CallCaptureShower interface {
	ShowRequest(func(string, ...interface{})) error
	ShowResponse(func(string, ...interface{})) error
}

var (
	_ CallCapturer      = (*tCapHTTP)(nil)
	_ CallCaptureShower = (*tCapHTTP)(nil)
	_ http.RoundTripper = (*tCapHTTP)(nil)
)

type tCapHTTP struct {
	showf    func(string, ...interface{})
	req, rep []byte

	har *HAR_Entry // FIXME: ditch HAR collector

	// Request/Response somewhat follow python's `requests` API

	/// request
	reqMethod  string
	reqURL     *url.URL
	reqHeaders map[string][]string
	reqHasBody bool
	reqBody    []byte
	reqJSON    interface{}
	/// reply
	repErr     string
	repStatus  int
	repReason  string
	repHeaders map[string][]string
	repHasBody bool
	repBody    []byte
	repJSON    interface{}

	elapsed time.Duration
	// TODO: pick from these
	// %{content_type} shows the Content-Type of the requested document, if there was any.
	// %{filename_effective} shows the ultimate filename that curl writes out to. This is only meaningful if curl is told to write to a file with the --remote-name or --output option. It's most useful in combination with the --remote-header-name option.
	// %{ftp_entry_path} shows the initial path curl ended up in when logging on to the remote FTP server.
	// %{response_code} shows the numerical response code that was found in the last transfer.
	// %{http_connect} shows the numerical code that was found in the last response (from a proxy) to a curl CONNECT request.
	// %{local_ip} shows the IP address of the local end of the most recently done connection—can be either IPv4 or IPv6
	// %{local_port} shows the local port number of the most recently made connection
	// %{num_connects} shows the number of new connects made in the recent transfer.
	// %{num_redirects} shows the number of redirects that were followed in the request.
	// %{redirect_url} shows the actual URL a redirect would take you to when an HTTP request was made without -L to follow redirects.
	// %{remote_ip} shows the remote IP address of the most recently made connection—can be either IPv4 or IPv6.
	// %{remote_port} shows the remote port number of the most recently made connection.
	// %{size_download} shows the total number of bytes that were downloaded.
	// %{size_header} shows the total number of bytes of the downloaded headers.
	// %{size_request} shows the total number of bytes that were sent in the HTTP request.
	// %{size_upload} shows the total number of bytes that were uploaded.
	// %{speed_download} shows the average download speed that curl measured for the complete download in bytes per second.
	// %{speed_upload} shows the average upload speed that curl measured for the complete upload in bytes per second.
	// %{ssl_verify_result} shows the result of the SSL peer certificate verification that was requested. 0 means the verification was successful.
	// %{time_appconnect} shows the time, in seconds, it took from the start until the SSL/SSH/etc connect/handshake to the remote host was completed.
	// %{time_connect} shows the time, in seconds, it took from the start until the TCP connect to the remote host (or proxy) was completed.
	// %{time_namelookup} shows the time, in seconds, it took from the start until the name resolving was completed.
	// %{time_pretransfer} shows the time, in seconds, it took from the start until the file transfer was just about to begin. This includes all pre-transfer commands and negotiations that are specific to the particular protocol(s) involved.
	// %{time_redirect} shows the time, in seconds, it took for all redirection steps including name lookup, connect, pre-transfer and transfer before the final transaction was started. time_redirect shows the complete execution time for multiple redirections.
	// %{time_starttransfer} shows the time, in seconds, it took from the start until the first byte was just about to be transferred. This includes time_pretransfer and also the time the server needed to calculate the result.
	// %{time_total} shows the total time, in seconds, that the full operation lasted. The time will be displayed with millisecond resolution.
	// %{url_effective} shows the URL that was fetched last. This is particularly meaningful if you have told curl to follow Location: headers (with -L).

	// FIXME: not sure about this
	firstChecks []namedLambda
	matchedSID  sid
}

type namedLambda struct {
	name   string
	lambda CheckerFunc
}

func (c *tCapHTTP) CheckFirst() (string, CheckerFunc) {
	var nameAndLambda namedLambda
	nameAndLambda, c.firstChecks = c.firstChecks[0], c.firstChecks[1:]
	return nameAndLambda.name, nameAndLambda.lambda
}

func newHTTPTCap(showf func(string, ...interface{})) *tCapHTTP {
	c := &tCapHTTP{
		showf: showf,
	}
	c.firstChecks = []namedLambda{
		{"HTTP code", c.checkFirstHTTPCode},
		{"valid JSON response", c.checkFirstValidJSONResponse},
		{"response validates schema", c.checkFirstValidatesJSONSchema},
		{"", nil},
	}
	return c
}

func (c *tCapHTTP) checkFirstHTTPCode(mnk *Monkey) (s string, f []string) {
	endpoint := mnk.Vald.Spec.Endpoints[mnk.eid].GetJson()
	var ok bool
	// TODO: handle 1,2,3,4,5,XXX
	// TODO: think about overflow
	if c.matchedSID, ok = endpoint.Outputs[uint32(c.repStatus)]; !ok {
		f = append(f, fmt.Sprintf("unexpected HTTP code '%d'", c.repStatus))
		return
	}
	s = "HTTP code checked"
	return
}

func (c *tCapHTTP) checkFirstValidJSONResponse(mnk *Monkey) (s string, f []string) {
	if !c.repHasBody {
		f = append(f, "response body is empty")
		return
	}

	// TODO: get Unmarshal error of request() method & return it
	s = "response is valid JSON"
	return
}

func (c *tCapHTTP) checkFirstValidatesJSONSchema(mnk *Monkey) (s string, f []string) {
	if errs := mnk.Vald.Spec.Schemas.Validate(c.matchedSID, c.repJSON); len(errs) != 0 {
		f = errs
		return
	}
	s = "response validates JSON Schema"
	return
}

func (c *tCapHTTP) ShowRequest(showf func(string, ...interface{})) error {
	showf("%s", c.req)
	return nil
}

func (c *tCapHTTP) ShowResponse(showf func(string, ...interface{})) error {
	if c.rep == nil {
		return errors.New("response is unset")
	}
	showf("%s", c.rep)
	return nil
}

func (c *tCapHTTP) Request() map[string]interface{} {
	m := map[string]interface{}{
		"method":  c.reqMethod,
		"url":     c.reqURL.String(),
		"headers": c.reqHeaders,
		// "content" as bytes?
	}
	if c.reqHasBody {
		m["json"] = c.reqJSON
	}
	// TODO? Response *Response
	// Response is the redirect response which caused this request
	// to be created. This field is only populated during client
	// redirects.
	return m
}

func (c *tCapHTTP) Response() map[string]interface{} {
	m := map[string]interface{}{
		"request": c.Request(),
		// FIXME: "error": c.repErr,
		"status_code": c.repStatus,
		"reason":      c.repReason,
		"headers":     c.repHeaders,
		// "content" as bytes?
		// "history" :: []Rep (redirects)?
	}
	if c.repHasBody {
		m["json"] = c.repJSON
	}
	// TODO? TLS *tls.ConnectionState
	// TLS contains information about the TLS connection on which the
	// response was received. It is nil for unencrypted responses.
	// The pointer is shared between responses and should not be
	// modified.
	return m
}

func (c *tCapHTTP) request(r *http.Request) (err error) {
	c.reqMethod = r.Method
	c.reqURL = cloneURL(r.URL)

	c.reqHeaders = cloneHeader(r.Header)
	if _, ok := c.reqHeaders[headerContentLength]; ok {
		c.reqHeaders[headerContentLength] = []string{strconv.FormatInt(r.ContentLength, 10)}
	}
	if _, ok := c.reqHeaders[headerTransferEncoding]; ok {
		c.reqHeaders[headerTransferEncoding] = r.TransferEncoding
	}
	if _, ok := c.reqHeaders[headerHost]; ok {
		c.reqHeaders[headerHost] = []string{r.Host}
	}

	if r.Body != nil {
		if c.reqBody, err = ioutil.ReadAll(r.Body); err != nil {
			return
		}
		r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewReader(c.reqBody))
		if err = json.Unmarshal(c.reqBody, &c.reqJSON); err != nil {
			return
		}
		c.reqHasBody = true
	}

	return
}

func (c *tCapHTTP) response(r *http.Response) (err error) {
	// FIXME c.repErr
	c.repStatus = r.StatusCode
	c.repReason = r.Status

	c.repHeaders = cloneHeader(r.Header)
	if _, ok := c.repHeaders[headerContentLength]; ok {
		c.repHeaders[headerContentLength] = []string{strconv.FormatInt(r.ContentLength, 10)}
	}
	if _, ok := c.repHeaders[headerTransferEncoding]; ok {
		c.repHeaders[headerTransferEncoding] = r.TransferEncoding
	}

	if r.Body != nil {
		if c.repBody, err = ioutil.ReadAll(r.Body); err != nil {
			return
		}
		r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewReader(c.repBody))
		if err = json.Unmarshal(c.repBody, &c.repJSON); err != nil {
			return
		}
		c.repHasBody = true
	}

	if c.rep, err = httputil.DumpResponse(r, false); err != nil {
		return
	}
	// TODO: move httputil.DumpResponse to Response() method

	return
}

func (c *tCapHTTP) RoundTrip(req *http.Request) (rep *http.Response, err error) {
	// FIXME: should we really do json decoding here + encoding as well?

	if err = c.request(req); err != nil {
		return
	}

	start := time.Now()
	if rep, err = http.DefaultTransport.RoundTrip(req); err != nil {
		return
	}
	elapsed := time.Since(start)
	c.elapsed = elapsed

	if c.har, err = harEntry(req, rep, elapsed); err != nil {
		return
	}

	err = c.response(rep)
	return
}

// FIXME: remove this global by attaching to OpenAPIv3 modeler
var tcap *tCapHTTP

func (act *ReqDoCall) exec(mnk *Monkey) (err error) {
	mnk.progress.state("🙈")
	mnk.eid = act.EID

	tcap = newHTTPTCap(func(format string, s ...interface{}) {
		// TODO: prepend with 2-space indentation (somehow doesn't work)
		mnk.progress.showf(format, s)
	})
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

func (c *tCapHTTP) makeRequest(harReq *HAR_Request, host string) (nxt *RepCallDone, err error) {
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

	maybeUpdateUserAgentHeader(req)
	// NOTE: forces Request.Write to use req.URL.Host
	req.Host = ""
	req.URL.Scheme = configured.Scheme
	req.URL.Host = configured.Host

	log.Println("[NFO] ▼", harReq)
	// TODO: output `curl` requests when showing counterexample
	//   https://github.com/sethgrid/gencurl
	//   https://github.com/moul/http2curl
	// FIXME: info output in `curl` style with timings
	if c.req, err = httputil.DumpRequestOut(req, false); err != nil {
		return
	}
	// TODO: move httputil.DumpRequestOut to Request() method
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
		nxt.Response = c.har
		nxt.Success = true
	} else {
		c.repErr = err.Error()
		log.Println("[NFO] ▲", c.repErr)
		nxt.Reason = c.repErr
		nxt.Failure = true
	}

	if err = c.ShowResponse(c.showf); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func maybeUpdateUserAgentHeader(r *http.Request) {
	if hs, ok := r.Header[headerUserAgent]; ok {
		replace := false
		for _, h := range hs {
			if strings.HasPrefix(h, "FuzzyMonkey.co/") {
				replace = true
			}
		}
		if replace {
			r.Header[headerUserAgent] = []string{binTitle}
		}
	}
}

func cloneHeader(src http.Header) (dst http.Header) {
	if src == nil {
		return
	}
	dst = make(http.Header, len(src))
	for h, hs := range src {
		if hs == nil {
			dst[h] = nil
		} else {
			values := make([]string, len(hs))
			copy(values, hs)
			dst[h] = values
		}
	}
	return
}

// https://github.com/golang/go/blob/2c67cdf7cf59a685f3a5e705b6be85f32285acec/src/net/http/clone.go#L22
func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := new(url.URL)
	*u2 = *u
	if u.User != nil {
		u2.User = new(url.Userinfo)
		*u2.User = *u.User
	}
	return u2
}
