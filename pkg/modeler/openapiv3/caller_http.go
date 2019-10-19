package modeler_openapiv3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	// "github.com/FuzzyMonkeyCo/monkey/pkg/runtime"
	"github.com/gogo/protobuf/types"
)

var (
	headerContentLength    = http.CanonicalHeaderKey("Content-Length")
	headerHost             = http.CanonicalHeaderKey("Host")
	headerTransferEncoding = http.CanonicalHeaderKey("Transfer-Encoding")
	headerUserAgent        = http.CanonicalHeaderKey("User-Agent")
)

var (
	_ modeler.Caller        = (*tCapHTTP)(nil)
	_ modeler.CaptureShower = (*tCapHTTP)(nil)
	_ http.RoundTripper     = (*tCapHTTP)(nil)
)

type tCapHTTP struct {
	showf    func(string, ...interface{})
	req, rep []byte

	httpReq  *http.Request
	reqProto *fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_
	repProto *fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_

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

func (c *tCapHTTP) ToProto() *fm.Clt_Msg_CallResponseRaw {
	return &fm.Clt_Msg_CallResponseRaw{}
}

type namedLambda struct {
	name   string
	lambda modeler.CheckerFunc
}

func (c *tCapHTTP) CheckFirst() (string, modeler.CheckerFunc) {
	if len(c.firstChecks) == 0 {
		return "", nil
	}
	var nameAndLambda namedLambda
	nameAndLambda, c.firstChecks = c.firstChecks[0], c.firstChecks[1:]
	return nameAndLambda.name, nameAndLambda.lambda
}

func (m *oa3) NewCaller(msg *fm.Srv_Msg_Call, showf func(string, ...interface{})) (*tCapHTTP, error) {
	m.tcap = &tCapHTTP{
		showf: showf,
	}
	if err := m.tcap.callinputProtoToHttpReqAndReqStructWithHostAndUA(msg); err != nil {
		return nil, err
	}

	m.tcap.firstChecks = []namedLambda{
		{"HTTP code", m.checkFirstHTTPCode(msg.GetEId())},
		{"valid JSON response", m.checkFirstValidJSONResponse},
		{"response validates schema", m.checkFirstValidatesJSONSchema},
	}
	return m.tcap, nil
}

func (m *oa3) checkFirstHTTPCode(eId uint32) modeler.CheckerFunc {
	return func() (s string, f []string) {
		endpoint := m.vald.Spec.Endpoints[eId].GetJson()
		var ok bool
		// TODO: handle 1,2,3,4,5,XXX
		// TODO: think about overflow
		if m.tcap.matchedSID, ok = endpoint.Outputs[uint32(m.tcap.repStatus)]; !ok {
			f = append(f, fmt.Sprintf("unexpected HTTP code '%d'", m.tcap.repStatus))
			return
		}
		s = "HTTP code checked"
		return
	}
}

func (m *oa3) checkFirstValidJSONResponse() (s string, f []string) {
	if !m.tcap.repHasBody {
		f = append(f, "response body is empty")
		return
	}

	// TODO: get Unmarshal error of request() method & return it
	s = "response is valid JSON"
	return
}

func (m *oa3) checkFirstValidatesJSONSchema() (s string, f []string) {
	if errs := m.vald.Validate(m.tcap.matchedSID, m.tcap.repJSON); len(errs) != 0 {
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

func (c *tCapHTTP) Request() *types.Struct {
	s := types.Struct{
		Fields: map[string]*types.Value{
			"method":  {Kind: &types.Value_StringValue{c.reqMethod}},
			"url":     {Kind: &types.Value_StringValue{c.reqURL.String()}},
			"headers": {Kind: &types.Value_ListValue{c.reqHeaders}},
			// "content" as bytes?
		},
	}
	if c.reqHasBody {
		s.Fields["json"] = &types.Value{Kind: &types.Value_StringValue{c.reqJSON}}
	}
	// TODO? Response *Response
	// Response is the redirect response which caused this request
	// to be created. This field is only populated during client
	// redirects.
	return s
}

func (c *tCapHTTP) Response() *types.Struct {
	s := types.Struct{
		Fields: map[string]*types.Value{
			"request": {Kind: c.Request()},
			// FIXME: "error": c.repErr,
			"status_code": {Kind: &types.Value_StringValue{c.repStatus}},
			"reason":      {Kind: &types.Value_StringValue{c.repReason}},
			"headers":     {Kind: &types.Value_ListValue{c.repHeaders}},
			// "content" as bytes?
			// "history" :: []Rep (redirects)?
		},
	}
	if c.repHasBody {
		s.Fields["json"] = c.repJSON
	}
	// TODO? TLS *tls.ConnectionState
	// TLS contains information about the TLS connection on which the
	// response was received. It is nil for unencrypted responses.
	// The pointer is shared between responses and should not be
	// modified.
	return s
}

func (c *tCapHTTP) request(r *http.Request) (err error) {
	c.reqProto = &fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_{
		Method: r.Method,
		Url: r.URL.String(),
	}

	reqHeaders := cloneHeader(r.Header)
	for key, values := range reqHeaders {
		switch key {
		case headerContentLength:
			v := strconv.FormatInt(r.ContentLength, 10)
			reqHeaders[headerContentLength] = &types.Value_ListValue{Values: {v}}
		case headerTransferEncoding:
			v := r.TransferEncoding
			reqHeaders[headerTransferEncoding] = &types.Value_ListValue{
				&types.Value_StringValue{v}
		case headerHost:
			v := r.Host
			reqHeaders[headerHost] = []string{r.Host}
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

	t := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			// TODO: snap the envs that ProxyFromEnvironment reads
			log.Println("[NFO] HTTP proxying is work in progress...")
			return nil, nil
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		// ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	start := time.Now()
	if rep, err = t.RoundTrip(req); err != nil {
		return
	}
	elapsed := time.Since(start)

	if c.reqProto, err = reqToProto(req, elapsed); err != nil {
		return
	}
	if c.repProto, err = repToProto(rep, elapsed); err != nil {
		return
	}

	err = c.response(rep)
	return
}

func (c *tCapHTTP) callinputProtoToHttpReqAndReqStructWithHostAndUA(msg *fm.Srv_Msg_Call) (err error) {
	input := msg.GetHttpRequest()
	if body := input.GetBody(); len(body) != 0 {
		b := bytes.NewReader(body)
		c.httpReq, err = http.NewRequest(input.GetMethod(), input.GetUrl(), b)
	} else {
		c.httpReq, err = http.NewRequest(input.GetMethod(), input.GetUrl(), nil)
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for key, values := range input.GetHeaders() {
		for _, value := range values.GetValues() {
			c.httpReq.Header.Add(key, value)
		}
	}
	return
}

func (c *tCapHTTP) Do(ctx context.Context) error {
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

	var r *http.Response
	r, err = (&http.Client{
		Transport: c,
	}).Do(req)

	nxt = &RepCallDone{TsDiff: uint64(c.elapsed)}
	if err == nil {
		r.Body.Close()
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
			// r.Header[headerUserAgent] = []string{runtime.BinTitle()}
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
