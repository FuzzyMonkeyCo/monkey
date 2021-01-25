package openapiv3

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime/ctxvalues"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
)

var (
	headerContentLength    = http.CanonicalHeaderKey("Content-Length")
	headerHost             = http.CanonicalHeaderKey("Host")
	headerTransferEncoding = http.CanonicalHeaderKey("Transfer-Encoding")
	headerUserAgent        = http.CanonicalHeaderKey("User-Agent")
)

var (
	_ modeler.Caller    = (*tCapHTTP)(nil)
	_ http.RoundTripper = (*tCapHTTP)(nil)
)

type tCapHTTP struct {
	showf       modeler.ShowFunc
	beforeDoErr error
	doErr       error
	rep         []byte

	endpoint        *fm.EndpointJSON
	matchedOutputID uint32
	matchedSID      sid
	matchedHTTPCode bool

	checks []namedLambda

	httpReq  *http.Request
	reqProto *fm.Clt_CallRequestRaw_Input_HttpRequest
	repProto *fm.Clt_CallResponseRaw_Output_HttpResponse

	reqValue, repValue *types.Value
	repJSONRaw         interface{}
	repJSONErr         error

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
}

func (c *tCapHTTP) ToProto() (i *fm.Clt_CallRequestRaw, o *fm.Clt_CallResponseRaw) {
	i = &fm.Clt_CallRequestRaw{
		Input: &fm.Clt_CallRequestRaw_Input{
			Input: &fm.Clt_CallRequestRaw_Input_HttpRequest_{
				HttpRequest: c.reqProto,
			}}}
	o = &fm.Clt_CallResponseRaw{
		OutputId: c.matchedOutputID,
		Output: &fm.Clt_CallResponseRaw_Output{
			Output: &fm.Clt_CallResponseRaw_Output_HttpResponse_{
				HttpResponse: c.repProto,
			}}}
	return
}

type namedLambda struct {
	name   string
	lambda modeler.CheckerFunc
}

func (c *tCapHTTP) NextCallerCheck() (string, modeler.CheckerFunc) {
	if len(c.checks) == 0 {
		return "", nil
	}
	var nameAndLambda namedLambda
	nameAndLambda, c.checks = c.checks[0], c.checks[1:]
	return nameAndLambda.name, nameAndLambda.lambda
}

func (m *oa3) NewCaller(ctx context.Context, msg *fm.Srv_Call, showf modeler.ShowFunc) modeler.Caller {
	m.tcap = &tCapHTTP{
		showf:    showf,
		endpoint: m.vald.Spec.Endpoints[msg.GetEID()].GetJson(),
	}

	// Some things have to be computed before Do() gets called:
	err := m.callinputProtoToHTTPReqAndReqStructWithHostAndUA(ctx, msg)

	m.tcap.beforeDoErr = err
	m.tcap.checks = []namedLambda{
		{"creation of HTTP request", m.checkCreateHTTPReq},
		{"connection to server", m.checkConn},
		{"code < 500", m.checkNot5XX},
		//FIXME: when decoupling modeler/caller move these to modeler
		{"HTTP code", m.checkHTTPCode},
		//TODO: check media type matches spec here (Content-Type: application/json)
		{"valid JSON response", m.checkValidJSONResponse},
		{"response validates schema", m.checkValidatesJSONSchema},
	}
	return m.tcap
}

func (m *oa3) checkCreateHTTPReq() (s, skipped string, f []string) {
	if err := m.tcap.beforeDoErr; err != nil {
		f = append(f, "could not create HTTP request")
		f = append(f, err.Error())
		return
	}
	s = "request created"
	return
}

func (m *oa3) checkConn() (s, skipped string, f []string) {
	if err := m.tcap.doErr; err != nil {
		f = append(f, "communication with server could not be established")
		f = append(f, err.Error())
		return
	}
	s = "request sent"
	return
}

func (m *oa3) checkNot5XX() (s, skipped string, f []string) {
	if code := m.tcap.repProto.StatusCode; code >= 500 {
		f = append(f, fmt.Sprintf("server error: '%d'", code))
		return
	}
	s = "no server error"
	return
}

func (m *oa3) checkHTTPCode() (s, skipped string, f []string) {
	if m.tcap.matchedHTTPCode {
		s = "HTTP code checked"
	} else {
		code := m.tcap.repProto.StatusCode
		f = append(f, fmt.Sprintf("unexpected HTTP code '%d'", code))
	}
	return
}

func (m *oa3) checkValidJSONResponse() (s, skipped string, f []string) {
	if m.tcap.isRepBodyEmpty() {
		s = "response body is empty"
		return
	}

	if m.tcap.repJSONErr != nil {
		f = append(f, m.tcap.repJSONErr.Error())
		return
	}

	m.tcap.repValue = enumFromGo(m.tcap.repJSONRaw)
	m.tcap.repJSONRaw = nil
	s = "response is valid JSON"
	return
}

func (m *oa3) checkValidatesJSONSchema() (s, skipped string, f []string) {
	if m.tcap.matchedSID == 0 {
		skipped = "no JSON Schema specified for response"
		return
	}
	if m.tcap.isRepBodyEmpty() {
		skipped = "response body is empty"
		return
	}
	if errs := m.vald.Validate(m.tcap.matchedSID, m.tcap.repValue); len(errs) != 0 {
		f = errs
		return
	}
	s = "response validates JSON Schema"
	return
}

func (c *tCapHTTP) showRequest(req []byte) {
	for _, line := range bytes.Split(req, []byte{'\r', '\n'}) {
		c.showf("> %s", line)
		break
	}
}

func (c *tCapHTTP) showResponse() {
	if err := c.doErr; err != nil {
		c.showf("HTTP error: %s\n", err.Error())
		return
	}
	for _, line := range bytes.Split(c.rep, []byte{'\r', '\n'}) {
		c.showf("< %s", line)
		break
	}
}

func (c *tCapHTTP) Request() *types.Struct {
	if c.beforeDoErr != nil {
		return nil
	}
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
		vals := make([]*types.Value, 0, len(values))
		for _, val := range values {
			vals = append(vals, enumFromGo(val))
		}
		headers[key] = &types.Value{Kind: &types.Value_ListValue{
			ListValue: &types.ListValue{Values: vals}}}
	}
	s.Fields["headers"] = &types.Value{Kind: &types.Value_StructValue{
		StructValue: &types.Struct{Fields: headers}}}

	if c.reqProto.Body != nil {
		s.Fields["body"] = c.reqValue
	}
	// TODO? Response *Response
	// Response is the redirect response which caused this request
	// to be created. This field is only populated during client
	// redirects.
	return s
}

// Request/Response somewhat follow python's `requests` API

func (c *tCapHTTP) Response() *types.Struct {
	request := c.Request()
	if request == nil {
		return nil
	}
	s := &types.Struct{
		Fields: map[string]*types.Value{
			"request": {Kind: &types.Value_StructValue{StructValue: request}},
			// FIXME? "error": enumFromGo(c.repProto.Error),
			"status_code": enumFromGo(c.repProto.StatusCode),
			"reason":      enumFromGo(c.repProto.Reason),
			// "content" as bytes?
			// "history" :: []Rep (redirects)?
			"elapsed_ns": enumFromGo(c.repProto.ElapsedNs),
		},
	}

	headers := make(map[string]*types.Value, len(c.repProto.Headers))
	for key, values0 := range c.repProto.Headers {
		values := values0.GetValues()
		vals := make([]*types.Value, 0, len(values))
		for _, val := range values {
			vals = append(vals, enumFromGo(val))
		}
		headers[key] = &types.Value{Kind: &types.Value_ListValue{
			ListValue: &types.ListValue{Values: vals}}}
	}
	s.Fields["headers"] = &types.Value{Kind: &types.Value_StructValue{
		StructValue: &types.Struct{Fields: headers}}}

	if !c.isRepBodyEmpty() {
		s.Fields["body"] = c.repValue
	}
	// TODO? TLS *tls.ConnectionState
	// TLS contains information about the TLS connection on which the
	// response was received. It is nil for unencrypted responses.
	// The pointer is shared between responses and should not be
	// modified.
	return s
}

// Records the request as actually performed: from http.Request.
func (c *tCapHTTP) request(r *http.Request) (err error) {
	c.reqProto = &fm.Clt_CallRequestRaw_Input_HttpRequest{
		Method: r.Method,
		Url:    r.URL.String(),
	}

	headers := fromReqHeader(r.Header)
	for key := range headers {
		switch key {
		case headerContentLength:
			headers[key] = &fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: []string{strconv.FormatInt(r.ContentLength, 10)},
			}
		case headerTransferEncoding:
			headers[key] = &fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: r.TransferEncoding,
			}
		case headerHost:
			headers[key] = &fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: []string{r.Host},
			}
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
		var jsn interface{}
		if e := json.Unmarshal(c.reqProto.Body, &jsn); e != nil {
			log.Println("[NFO] request wasn't proper JSON:", e)
		} else {
			c.reqValue = enumFromGo(jsn)
		}
	}

	return
}

func (c *tCapHTTP) response(r *http.Response, e error) (err error) {
	if e != nil {
		c.repProto.Error = e.Error()
		c.rep = []byte(e.Error())
		return
	}
	c.repProto.StatusCode = uint32(r.StatusCode)
	c.repProto.Reason = r.Status

	headers := fromRepHeader(r.Header)
	for key := range headers {
		switch key {
		case headerContentLength:
			headers[key] = &fm.Clt_CallResponseRaw_Output_HttpResponse_HeaderValues{
				Values: []string{strconv.FormatInt(r.ContentLength, 10)},
			}
		case headerTransferEncoding:
			headers[key] = &fm.Clt_CallResponseRaw_Output_HttpResponse_HeaderValues{
				Values: r.TransferEncoding,
			}
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
		if !c.isRepBodyEmpty() {
			c.repJSONErr = json.Unmarshal(c.repProto.Body, &c.repJSONRaw)
		}
	}

	func() {
		var ok bool
		outputID := c.repProto.StatusCode
		if c.matchedSID, ok = c.endpoint.Outputs[outputID]; !ok {
			outputID = fromStatusCode(outputID)
			if c.matchedSID, ok = c.endpoint.Outputs[outputID]; !ok {
				outputID = 0
				if c.matchedSID, ok = c.endpoint.Outputs[outputID]; !ok {
					return
				}
			}
		}
		c.matchedOutputID = outputID
		c.matchedHTTPCode = true
	}()

	if c.rep, err = httputil.DumpResponse(r, false); err != nil {
		return
	}
	// TODO: move httputil.DumpResponse to Response() method

	return
}

func (c *tCapHTTP) isRepBodyEmpty() bool { return len(c.repProto.Body) == 0 }

func (c *tCapHTTP) RoundTrip(req *http.Request) (rep *http.Response, err error) {
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
	rep, err = t.RoundTrip(req)
	c.repProto = &fm.Clt_CallResponseRaw_Output_HttpResponse{
		ElapsedNs: time.Since(start).Nanoseconds(),
	}

	if e := c.response(rep, err); e != nil {
		if err == nil {
			err = e
		}
	}
	return
}

func (m *oa3) callinputProtoToHTTPReqAndReqStructWithHostAndUA(ctx context.Context, msg *fm.Srv_Call) (err error) {
	input := msg.GetInput().GetHttpRequest()

	if body := input.GetBody(); body != nil {
		buf := &bytes.Buffer{}
		if err = (&jsonpb.Marshaler{}).Marshal(buf, body); err != nil {
			log.Println("[ERR]", err)
			return
		}
		m.tcap.httpReq, err = http.NewRequest(input.GetMethod(), input.GetUrl(), buf)
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

	if authz := m.HeaderAuthorization; authz != "" {
		m.tcap.httpReq.Header.Add("Authorization", authz)
	}

	m.tcap.httpReq.Header.Set(headerUserAgent, ctx.Value(ctxvalues.UserAgent).(string))

	if host := m.Host; host != "" {
		var configured *url.URL
		if configured, err = url.ParseRequestURI(host); err != nil {
			log.Println("[ERR]", err)
			return
		}

		// NOTE: forces Request.Write to use URL.Host
		m.tcap.httpReq.Host = ""
		m.tcap.httpReq.URL.Scheme = configured.Scheme
		m.tcap.httpReq.URL.Host = configured.Host
	}

	return
}

func (c *tCapHTTP) Do(ctx context.Context) {
	if c.beforeDoErr != nil {
		// An error happened during NewCaller. Report is delayed until
		// the call to checkCreateHTTPReq.
		// c.repProto.Error cannot be set as the rest of the Request()/Response()
		// protos would be empty. Instead we have them return nil.
		return
	}

	// TODO: output `curl` requests when showing counterexample
	//   https://github.com/sethgrid/gencurl
	//   https://github.com/moul/http2curl
	// FIXME: info output in `curl` style with timings
	var err error
	var req []byte
	if req, err = httputil.DumpRequestOut(c.httpReq, false); err != nil {
		log.Println("[ERR]", err)
		return
	}
	// TODO: move httputil.DumpRequestOut to Request() method
	c.showRequest(req)

	var r *http.Response
	if r, c.doErr = (&http.Client{Transport: c}).Do(c.httpReq); c.doErr == nil {
		r.Body.Close()
	}

	c.showResponse()
	return
}

func fromReqHeader(src http.Header) map[string]*fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues {
	if src == nil {
		return nil
	}
	dst := make(map[string]*fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues, len(src))
	for h, hs := range src {
		if len(hs) != 0 {
			dst[h] = &fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues{
				Values: hs,
			}
		}
	}
	return dst
}

func fromRepHeader(src http.Header) map[string]*fm.Clt_CallResponseRaw_Output_HttpResponse_HeaderValues {
	if src == nil {
		return nil
	}
	dst := make(map[string]*fm.Clt_CallResponseRaw_Output_HttpResponse_HeaderValues, len(src))
	for h, hs := range src {
		if len(hs) != 0 {
			dst[h] = &fm.Clt_CallResponseRaw_Output_HttpResponse_HeaderValues{
				Values: hs,
			}
		}
	}
	return dst
}
