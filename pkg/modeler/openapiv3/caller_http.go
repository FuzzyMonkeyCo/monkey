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
	reqProto *fm.Clt_Msg_CallRequestRaw_Input_HttpRequest
	repProto *fm.Clt_Msg_CallResponseRaw_Output_HttpResponse

	reqJSON, repJSON *types.Value

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
	return &fm.Clt_Msg_CallResponseRaw{Output: &fm.Clt_Msg_CallResponseRaw_Output{
		Output: &fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_{
			HttpResponse: c.repProto,
		}}}
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

func (m *oa3) NewCaller(msg *fm.Srv_Msg_Call, showf func(string, ...interface{})) (modeler.Caller, error) {
	m.tcap = &tCapHTTP{
		showf: showf,
	}
	if err := m.callinputProtoToHttpReqAndReqStructWithHostAndUA(msg); err != nil {
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
		code := m.tcap.repProto.StatusCode
		var ok bool
		// TODO: handle 1,2,3,4,5,XXX
		// TODO: think about overflow
		if m.tcap.matchedSID, ok = endpoint.Outputs[uint32(code)]; !ok {
			f = append(f, fmt.Sprintf("unexpected HTTP code '%d'", code))
			return
		}
		s = "HTTP code checked"
		return
	}
}

func (m *oa3) checkFirstValidJSONResponse() (s string, f []string) {
	// if m.tcap.repProto.Body != nil {
	// 	f = append(f, "response body is empty")
	// 	return
	// }

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
	s := &types.Struct{
		Fields: map[string]*types.Value{
			"method": enumFromGo(c.reqProto.Method),
			"url":    enumFromGo(c.reqProto.Url),
			// "content" as bytes?
		},
	}

	headers := make(map[string]*types.Value, len(c.reqProto.Headers))
	for key, values0 := range c.reqProto.Headers {
		values := values0.GetValues()
		vals := make([]*types.Value, len(values))
		for i, val := range values {
			vals[i] = enumFromGo(val)
		}
		headers[key] = &types.Value{Kind: &types.Value_ListValue{
			ListValue: &types.ListValue{Values: vals}}}
	}
	s.Fields["headers"] = &types.Value{Kind: &types.Value_StructValue{
		StructValue: &types.Struct{Fields: headers}}}

	if c.reqProto.Body != nil {
		s.Fields["json"] = c.reqJSON
	}
	// TODO? Response *Response
	// Response is the redirect response which caused this request
	// to be created. This field is only populated during client
	// redirects.
	return s
}

// Request/Response somewhat follow python's `requests` API

func (c *tCapHTTP) Response() *types.Struct {
	s := &types.Struct{
		Fields: map[string]*types.Value{
			"request": &types.Value{Kind: &types.Value_StructValue{c.Request()}},
			// FIXME? "error"
			"status_code": enumFromGo(c.repProto.StatusCode),
			"reason":      enumFromGo(c.repProto.Reason),
			// "content" as bytes?
			// "history" :: []Rep (redirects)?
		},
	}

	headers := make(map[string]*types.Value, len(c.repProto.Headers))
	for key, values0 := range c.repProto.Headers {
		values := values0.GetValues()
		vals := make([]*types.Value, len(values))
		for i, val := range values {
			vals[i] = enumFromGo(val)
		}
		headers[key] = &types.Value{Kind: &types.Value_ListValue{
			ListValue: &types.ListValue{Values: vals}}}
	}
	s.Fields["headers"] = &types.Value{Kind: &types.Value_StructValue{
		StructValue: &types.Struct{Fields: headers}}}

	if c.repProto.Body != nil {
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
	c.reqProto = &fm.Clt_Msg_CallRequestRaw_Input_HttpRequest{
		Method: r.Method,
		Url:    r.URL.String(),
	}

	headers := fromReqHeader(r.Header)
	for key, _ := range headers {
		switch key {
		case headerContentLength:
			headers[key] = &fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: []string{strconv.FormatInt(r.ContentLength, 10)},
			}
		case headerTransferEncoding:
			values := make([]string, len(r.TransferEncoding))
			copy(values, r.TransferEncoding)
			headers[key] = &fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: values,
			}
		case headerHost:
			headers[key] = &fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: []string{r.Host},
			}
		default:
		}
	}
	c.reqProto.Headers = headers

	if r.Body != nil {
		if c.reqProto.Body, err = ioutil.ReadAll(r.Body); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = r.Body.Close(); err != nil {
			log.Println("[ERR]", err)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(c.reqProto.Body))
		// TOOD: move decoding to one of CheckFirst
		var jsn interface{}
		if err = json.Unmarshal(c.reqProto.Body, &jsn); err != nil {
			log.Println("[ERR]", err)
			return
		}
		c.reqJSON = enumFromGo(jsn)
	}

	return
}

func (c *tCapHTTP) response(r *http.Response, elapsed time.Duration, e error) (err error) {
	c.repProto = &fm.Clt_Msg_CallResponseRaw_Output_HttpResponse{
		StatusCode: uint32(r.StatusCode), // TODO: check bounds
		Reason:     r.Status,
		Elapsed:    uint32(elapsed),
	}
	if e != nil {
		c.repProto.Error = e.Error()
	}

	headers := fromRepHeader(r.Header)
	for key, _ := range headers {
		switch key {
		case headerContentLength:
			headers[key] = &fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_HeaderValues{
				Values: []string{strconv.FormatInt(r.ContentLength, 10)},
			}
		case headerTransferEncoding:
			values := make([]string, len(r.TransferEncoding))
			copy(values, r.TransferEncoding)
			headers[key] = &fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_HeaderValues{
				Values: values,
			}
		default:
		}
	}
	c.repProto.Headers = headers

	if r.Body != nil {
		if c.repProto.Body, err = ioutil.ReadAll(r.Body); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = r.Body.Close(); err != nil {
			log.Println("[ERR]", err)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(c.repProto.Body))
		// TOOD: move decoding to one of CheckFirst
		var jsn interface{}
		if err = json.Unmarshal(c.repProto.Body, &jsn); err != nil {
			log.Println("[ERR]", err)
			return
		}
		c.repJSON = enumFromGo(jsn)
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

	callFailed := err != nil
	err = c.response(rep, elapsed, err)
	if callFailed {
		err = modeler.ErrCallFailed
	}
	return
}

func (m *oa3) callinputProtoToHttpReqAndReqStructWithHostAndUA(msg *fm.Srv_Msg_Call) (err error) {
	input := msg.GetInput().GetHttpRequest()
	if body := input.GetBody(); len(body) != 0 {
		b := bytes.NewReader(body)
		m.tcap.httpReq, err = http.NewRequest(input.GetMethod(), input.GetUrl(), b)
	} else {
		m.tcap.httpReq, err = http.NewRequest(input.GetMethod(), input.GetUrl(), nil)
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for key, values := range input.GetHeaders() {
		for _, value := range values.GetValues() {
			m.tcap.httpReq.Header.Add(key, value)
		}
	}

	if m.HeaderAuthorization != "" {
		m.tcap.httpReq.Header.Add("Authorization", m.HeaderAuthorization)
	}

	// m.tcap.httpReq.Header.Set(headerUserAgent, runtime.BinTitle())

	if host := m.Host; host != "" {
		configured, err := url.ParseRequestURI(host)
		if err != nil {
			log.Println("[ERR]", err)
			return err
		}

		// NOTE: forces Request.Write to use URL.Host
		m.tcap.httpReq.Host = ""
		m.tcap.httpReq.URL.Scheme = configured.Scheme
		m.tcap.httpReq.URL.Host = configured.Host
	}

	return
}

func (c *tCapHTTP) Do(ctx context.Context) (err error) {
	req := c.httpReq

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

	if err == nil {
		r.Body.Close()
	}

	if err = c.ShowResponse(c.showf); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func fromReqHeader(src http.Header) map[string]*fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_HeaderValues {
	if src == nil {
		return nil
	}
	dst := make(map[string]*fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_HeaderValues, len(src))
	for h, hs := range src {
		if len(hs) != 0 {
			vs := make([]string, len(hs))
			copy(vs, hs)
			dst[h] = &fm.Clt_Msg_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: vs,
			}
		}
	}
	return dst
}

func fromRepHeader(src http.Header) map[string]*fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_HeaderValues {
	if src == nil {
		return nil
	}
	dst := make(map[string]*fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_HeaderValues, len(src))
	for h, hs := range src {
		if len(hs) != 0 {
			vs := make([]string, len(hs))
			copy(vs, hs)
			dst[h] = &fm.Clt_Msg_CallResponseRaw_Output_HttpResponse_HeaderValues{
				Values: vs,
			}
		}
	}
	return dst
}
