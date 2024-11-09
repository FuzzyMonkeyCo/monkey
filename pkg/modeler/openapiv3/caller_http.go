package openapiv3

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime/ctxvalues"
)

var (
	headerAuthorization    = http.CanonicalHeaderKey("Authorization")
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
	shower              progresser.Shower
	buildHTTPRequestErr error
	doErr               error

	endpoint        *fm.EndpointJSON
	matchedOutputID uint32
	matchedSID      sid
	matchedHTTPCode bool

	checks []namedLambda

	httpReq          *http.Request
	repProto         *fm.Clt_CallResponseRaw_Output_HttpResponse
	repBodyDecodeErr error

	// TODO: pick from these and timings
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

// NewCaller creates a single-use modeler.Caller from a modeler.Interface instance.
func (m *oa3) NewCaller(ctx context.Context, msg *fm.Srv_Call, shower progresser.Shower) modeler.Caller {
	m.tcap = &tCapHTTP{
		shower:   shower,
		endpoint: m.vald.Spec.Endpoints[msg.GetEID()].GetJson(),
	}
	m.tcap.httpReq, m.tcap.buildHTTPRequestErr = m.buildHTTPRequest(ctx, msg)
	m.tcap.checks = m.callerChecks()
	return m.tcap
}

func (m *oa3) buildHTTPRequest(ctx context.Context, msg *fm.Srv_Call) (req *http.Request, err error) {
	input := msg.GetInput().GetHttpRequest()
	var r *http.Request

	if body := input.GetBody(); body != nil {
		var bodyBytes []byte
		if bodyBytes, err = protojson.Marshal(body); err != nil {
			log.Println("[ERR]", err)
			return
		}
		r, err = http.NewRequestWithContext(ctx, input.GetMethod(), input.GetUrl(), bytes.NewReader(bodyBytes))
	} else {
		r, err = http.NewRequestWithContext(ctx, input.GetMethod(), input.GetUrl(), nil)
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for _, kvs := range input.GetHeaders() {
		key := kvs.GetKey()
		for _, value := range kvs.GetValues() {
			r.Header.Add(key, value)
		}
	}

	r.Header.Set(headerUserAgent, ctx.Value(ctxvalues.XUserAgent).(string))

	if host := m.pb.Host; host != "" {
		var configured *url.URL
		if configured, err = url.ParseRequestURI(host); err != nil {
			log.Println("[ERR]", err)
			return
		}

		// NOTE: forces Request.Write to use URL.Host
		r.Host = ""
		r.URL.Scheme = configured.Scheme
		r.URL.Host = configured.Host
	}

	req = r
	return
}

// RequestProto returns call input as used by the client
func (c *tCapHTTP) RequestProto() (i *fm.Clt_CallRequestRaw) {
	i = &fm.Clt_CallRequestRaw{}
	if err := c.buildHTTPRequestErr; err != nil {
		i.Reason = strings.Split(err.Error(), "\n")
		return
	}
	reqProto, err := requestToProto(c.httpReq)
	if err != nil {
		i.Reason = strings.Split(err.Error(), "\n")
		return
	}
	i.Input = &fm.Clt_CallRequestRaw_Input{Input: &fm.Clt_CallRequestRaw_Input_HttpRequest_{
		HttpRequest: reqProto,
	}}
	return
}

// Records the request as actually performed: from http.Request.
func requestToProto(r *http.Request) (
	reqProto *fm.Clt_CallRequestRaw_Input_HttpRequest,
	err error,
) {
	reqProto = &fm.Clt_CallRequestRaw_Input_HttpRequest{
		Method: r.Method,
		Url:    r.URL.String(),
	}

	headerNames := make([]string, 0, len(r.Header))
	for key := range r.Header {
		headerNames = append(headerNames, key)
	}
	sort.Strings(headerNames)
	reqProto.Headers = make([]*fm.HeaderPair, 0, len(headerNames))
	for _, key := range headerNames {
		values := r.Header[key]
		switch key {
		case headerContentLength:
			newvalues := []string{strconv.FormatInt(r.ContentLength, 10)}
			log.Printf("[NFO] replacing %s headers %+v with %+v", key, values, newvalues)
			values = newvalues
		case headerTransferEncoding:
			newvalues := r.TransferEncoding
			log.Printf("[NFO] replacing %s headers %+v with %+v", key, values, newvalues)
			values = newvalues
		case headerHost:
			newvalues := []string{r.Host}
			log.Printf("[NFO] replacing %s headers %+v with %+v", key, values, newvalues)
			values = newvalues
		}
		reqProto.Headers = append(reqProto.Headers, &fm.HeaderPair{
			Key:    key,
			Values: values,
		})
	}

	if r.Body != nil {
		if reqProto.Body, err = ioutil.ReadAll(r.Body); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = r.Body.Close(); err != nil {
			log.Println("[ERR]", err)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(reqProto.Body))

		var x structpb.Value
		if e := protojson.Unmarshal(reqProto.Body, &x); e != nil {
			log.Println("[NFO] request body could not be decoded:", e)
		} else {
			reqProto.BodyDecoded = &x
		}
	}

	return
}

// Do sends the request and waits for the response
func (c *tCapHTTP) Do(ctx context.Context) {
	br := []byte{'\r', '\n'}
	req, err := httputil.DumpRequestOut(c.httpReq, false)
	if err != nil {
		log.Println("[ERR]", err)
		req = []byte(err.Error())
	}
	for _, line := range bytes.Split(req, br) {
		c.shower.Printf("> %s", line)
		break
	}

	var rep []byte
	var r *http.Response
	if r, c.doErr = (&http.Client{Transport: c}).Do(c.httpReq); c.doErr != nil {
		rep = []byte(fmt.Sprintf("HTTP error: %s", c.doErr.Error()))
	} else {
		r.Body.Close()
		if rep, err = httputil.DumpResponse(r, false); err != nil {
			log.Println("[ERR]", err)
			rep = []byte(err.Error())
		}
	}
	for _, line := range bytes.Split(rep, br) {
		c.shower.Printf("< %s\n", line)
		break
	}
	return
}

func (c *tCapHTTP) RoundTrip(req *http.Request) (rep *http.Response, err error) {
	// TODO: stricter/smaller timeouts https://pkg.go.dev/github.com/asecurityteam/transport#Option
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		// TODO: snap the envs that ProxyFromEnvironment reads
		log.Println("[NFO] HTTP proxying is work in progress...")
		return nil, nil
	}
	t.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext
	// t.ForceAttemptHTTP2 = true
	t.MaxIdleConns = 100
	t.IdleConnTimeout = 90 * time.Second
	t.TLSHandshakeTimeout = 10 * time.Second
	t.ExpectContinueTimeout = 1 * time.Second
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: os.Getenv("FUZZYMONKEY_SSL_NO_VERIFY") == "1"}

	start := time.Now()
	rep, err = t.RoundTrip(req)
	c.repProto = &fm.Clt_CallResponseRaw_Output_HttpResponse{
		ElapsedNs: time.Since(start).Nanoseconds(),
	}
	if err != nil {
		c.repProto.Error = err.Error()
		return
	}
	err = c.responseToProto(rep)
	return
}

// ResponseProto returns call output as received by the client
func (c *tCapHTTP) ResponseProto() *fm.Clt_CallResponseRaw {
	return &fm.Clt_CallResponseRaw{
		OutputId: c.matchedOutputID,
		Output: &fm.Clt_CallResponseRaw_Output{
			Output: &fm.Clt_CallResponseRaw_Output_HttpResponse_{
				HttpResponse: c.repProto,
			}}}
}

func (c *tCapHTTP) responseToProto(r *http.Response) (err error) {
	c.repProto.StatusCode = uint32(r.StatusCode)
	c.repProto.Reason = r.Status

	headerNames := make([]string, 0, len(r.Header))
	for key := range r.Header {
		headerNames = append(headerNames, key)
	}
	sort.Strings(headerNames)
	c.repProto.Headers = make([]*fm.HeaderPair, 0, len(headerNames))
	for _, key := range headerNames {
		values := r.Header[key]
		switch key {
		case headerContentLength:
			newvalues := []string{strconv.FormatInt(r.ContentLength, 10)}
			log.Printf("[NFO] replacing %s headers %+v with %+v", key, values, newvalues)
			values = newvalues
		case headerTransferEncoding:
			newvalues := r.TransferEncoding
			log.Printf("[NFO] replacing %s headers %+v with %+v", key, values, newvalues)
			values = newvalues
		}
		c.repProto.Headers = append(c.repProto.Headers, &fm.HeaderPair{
			Key:    key,
			Values: values,
		})
	}

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

		var x structpb.Value
		if e := protojson.Unmarshal(c.repProto.Body, &x); e != nil {
			log.Println("[NFO] response body could not be decoded:", e)
			c.repBodyDecodeErr = e
		} else {
			c.repProto.BodyDecoded = &x
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

	// TODO? redirects with Response *Response
	// Response is the redirect response which caused this request
	// to be created. This field is only populated during client
	// redirects.

	// TODO? TLS *tls.ConnectionState
	// TLS contains information about the TLS connection on which the
	// response was received. It is nil for unencrypted responses.
	// The pointer is shared between responses and should not be
	// modified.

	return
}

// NextCallerCheck returns ("",nil) when out of checks to run.
// Otherwise it returns named checks inherent to the caller.
func (c *tCapHTTP) NextCallerCheck() (string, modeler.CheckerFunc) {
	if len(c.checks) == 0 {
		return "", nil
	}
	var nameAndLambda namedLambda
	nameAndLambda, c.checks = c.checks[0], c.checks[1:]
	return nameAndLambda.name, nameAndLambda.lambda
}
