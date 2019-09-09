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
		log.Printf("[NFO] checking %d user properties", len(userRTLang.Triggers))
		response := slValueFromHAR(act.Response)
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
							if userRTLang.ModelState, ok = newModelState.(*modelState); ok {
								checkN.Success = true
								mnk.progress.checkPassed(checkN.Details[0])
							} else {
								checkN.Failure = true
								err = fmt.Errorf("expected action to return a ModelState, got: %v", newModelState)
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

func slValueFromHAR(entry *HAR_Entry) starlark.Value {

	// type Request struct {
	//     // Method specifies the HTTP method (GET, POST, PUT, etc.).
	//     // For client requests, an empty string means GET.
	//     //
	//     // Go's HTTP client does not support sending a request with
	//     // the CONNECT method. See the documentation on Transport for
	//     // details.
	//     Method string

	//     // URL specifies either the URI being requested (for server
	//     // requests) or the URL to access (for client requests).
	//     //
	//     // For server requests, the URL is parsed from the URI
	//     // supplied on the Request-Line as stored in RequestURI.  For
	//     // most requests, fields other than Path and RawQuery will be
	//     // empty. (See RFC 7230, Section 5.3)
	//     //
	//     // For client requests, the URL's Host specifies the server to
	//     // connect to, while the Request's Host field optionally
	//     // specifies the Host header value to send in the HTTP
	//     // request.
	//     URL *url.URL

	//     // The protocol version for incoming server requests.
	//     //
	//     // For client requests, these fields are ignored. The HTTP
	//     // client code always uses either HTTP/1.1 or HTTP/2.
	//     // See the docs on Transport for details.
	//     Proto      string // "HTTP/1.0"
	//     ProtoMajor int    // 1
	//     ProtoMinor int    // 0

	//     // Header contains the request header fields either received
	//     // by the server or to be sent by the client.
	//     //
	//     // If a server received a request with header lines,
	//     //
	//     //	Host: example.com
	//     //	accept-encoding: gzip, deflate
	//     //	Accept-Language: en-us
	//     //	fOO: Bar
	//     //	foo: two
	//     //
	//     // then
	//     //
	//     //	Header = map[string][]string{
	//     //		"Accept-Encoding": {"gzip, deflate"},
	//     //		"Accept-Language": {"en-us"},
	//     //		"Foo": {"Bar", "two"},
	//     //	}
	//     //
	//     // For incoming requests, the Host header is promoted to the
	//     // Request.Host field and removed from the Header map.
	//     //
	//     // HTTP defines that header names are case-insensitive. The
	//     // request parser implements this by using CanonicalHeaderKey,
	//     // making the first character and any characters following a
	//     // hyphen uppercase and the rest lowercase.
	//     //
	//     // For client requests, certain headers such as Content-Length
	//     // and Connection are automatically written when needed and
	//     // values in Header may be ignored. See the documentation
	//     // for the Request.Write method.
	//     Header Header

	//     // Body is the request's body.
	//     //
	//     // For client requests, a nil body means the request has no
	//     // body, such as a GET request. The HTTP Client's Transport
	//     // is responsible for calling the Close method.
	//     //
	//     // For server requests, the Request Body is always non-nil
	//     // but will return EOF immediately when no body is present.
	//     // The Server will close the request body. The ServeHTTP
	//     // Handler does not need to.
	//     Body io.ReadCloser

	//     // GetBody defines an optional func to return a new copy of
	//     // Body. It is used for client requests when a redirect requires
	//     // reading the body more than once. Use of GetBody still
	//     // requires setting Body.
	//     //
	//     // For server requests, it is unused.
	//     GetBody func() (io.ReadCloser, error)

	//     // ContentLength records the length of the associated content.
	//     // The value -1 indicates that the length is unknown.
	//     // Values >= 0 indicate that the given number of bytes may
	//     // be read from Body.
	//     //
	//     // For client requests, a value of 0 with a non-nil Body is
	//     // also treated as unknown.
	//     ContentLength int64

	//     // TransferEncoding lists the transfer encodings from outermost to
	//     // innermost. An empty list denotes the "identity" encoding.
	//     // TransferEncoding can usually be ignored; chunked encoding is
	//     // automatically added and removed as necessary when sending and
	//     // receiving requests.
	//     TransferEncoding []string

	//     // Close indicates whether to close the connection after
	//     // replying to this request (for servers) or after sending this
	//     // request and reading its response (for clients).
	//     //
	//     // For server requests, the HTTP server handles this automatically
	//     // and this field is not needed by Handlers.
	//     //
	//     // For client requests, setting this field prevents re-use of
	//     // TCP connections between requests to the same hosts, as if
	//     // Transport.DisableKeepAlives were set.
	//     Close bool

	//     // For server requests, Host specifies the host on which the URL
	//     // is sought. Per RFC 7230, section 5.4, this is either the value
	//     // of the "Host" header or the host name given in the URL itself.
	//     // It may be of the form "host:port". For international domain
	//     // names, Host may be in Punycode or Unicode form. Use
	//     // golang.org/x/net/idna to convert it to either format if
	//     // needed.
	//     // To prevent DNS rebinding attacks, server Handlers should
	//     // validate that the Host header has a value for which the
	//     // Handler considers itself authoritative. The included
	//     // ServeMux supports patterns registered to particular host
	//     // names and thus protects its registered Handlers.
	//     //
	//     // For client requests, Host optionally overrides the Host
	//     // header to send. If empty, the Request.Write method uses
	//     // the value of URL.Host. Host may contain an international
	//     // domain name.
	//     Host string

	//     // Form contains the parsed form data, including both the URL
	//     // field's query parameters and the PATCH, POST, or PUT form data.
	//     // This field is only available after ParseForm is called.
	//     // The HTTP client ignores Form and uses Body instead.
	//     Form url.Values

	//     // PostForm contains the parsed form data from PATCH, POST
	//     // or PUT body parameters.
	//     //
	//     // This field is only available after ParseForm is called.
	//     // The HTTP client ignores PostForm and uses Body instead.
	//     PostForm url.Values

	//     // MultipartForm is the parsed multipart form, including file uploads.
	//     // This field is only available after ParseMultipartForm is called.
	//     // The HTTP client ignores MultipartForm and uses Body instead.
	//     MultipartForm *multipart.Form

	//     // Trailer specifies additional headers that are sent after the request
	//     // body.
	//     //
	//     // For server requests, the Trailer map initially contains only the
	//     // trailer keys, with nil values. (The client declares which trailers it
	//     // will later send.)  While the handler is reading from Body, it must
	//     // not reference Trailer. After reading from Body returns EOF, Trailer
	//     // can be read again and will contain non-nil values, if they were sent
	//     // by the client.
	//     //
	//     // For client requests, Trailer must be initialized to a map containing
	//     // the trailer keys to later send. The values may be nil or their final
	//     // values. The ContentLength must be 0 or -1, to send a chunked request.
	//     // After the HTTP request is sent the map values can be updated while
	//     // the request body is read. Once the body returns EOF, the caller must
	//     // not mutate Trailer.
	//     //
	//     // Few HTTP clients, servers, or proxies support HTTP trailers.
	//     Trailer Header

	//     // RemoteAddr allows HTTP servers and other software to record
	//     // the network address that sent the request, usually for
	//     // logging. This field is not filled in by ReadRequest and
	//     // has no defined format. The HTTP server in this package
	//     // sets RemoteAddr to an "IP:port" address before invoking a
	//     // handler.
	//     // This field is ignored by the HTTP client.
	//     RemoteAddr string

	//     // RequestURI is the unmodified request-target of the
	//     // Request-Line (RFC 7230, Section 3.1.1) as sent by the clientoo
	//     // to a server. Usually the URL field should be used instead.
	//     // It is an error to set this field in an HTTP client request.
	//     RequestURI string

	//     // TLS allows HTTP servers and other software to record
	//     // information about the TLS connection on which the request
	//     // was received. This field is not filled in by ReadRequest.
	//     // The HTTP server in this package sets the field for
	//     // TLS-enabled connections before invoking a handler;
	//     // otherwise it leaves the field nil.
	//     // This field is ignored by the HTTP client.
	//     TLS *tls.ConnectionState

	//     // Cancel is an optional channel whose closure indicates that the client
	//     // request should be regarded as canceled. Not all implementations of
	//     // RoundTripper may support Cancel.
	//     //
	//     // For server requests, this field is not applicable.
	//     //
	//     // Deprecated: Set the Request's context with NewRequestWithContext
	//     // instead. If a Request's Cancel field and context are both
	//     // set, it is undefined whether Cancel is respected.
	//     Cancel <-chan struct{}

	//     // Response is the redirect response which caused this request
	//     // to be created. This field is only populated during client
	//     // redirects.
	//     Response *Response
	//     // contains filtered or unexported fields
	// }

	/// Rep
	// status string "200 OK"
	// status_code int 200
	// proto string "HTTP/1.0"
	// proto_major int 1
	// proto_minor int 0
	// headers {string: [string]} {"Cookie": ["..."]}
	// body Value <>

	// Actually nevermind HAR, just copy requests' API here

	//     // Body represents the response body.
	//     //
	//     // The response body is streamed on demand as the Body field
	//     // is read. If the network connection fails or the server
	//     // terminates the response, Body.Read calls return an error.
	//     //
	//     // The http Client and Transport guarantee that Body is always
	//     // non-nil, even on responses without a body or responses with
	//     // a zero-length body. It is the caller's responsibility to
	//     // close Body. The default HTTP client's Transport may not
	//     // reuse HTTP/1.x "keep-alive" TCP connections if the Body is
	//     // not read to completion and closed.
	//     //
	//     // The Body is automatically dechunked if the server replied
	//     // with a "chunked" Transfer-Encoding.
	//     //
	//     // As of Go 1.12, the Body will also implement io.Writer
	//     // on a successful "101 Switching Protocols" response,
	//     // as used by WebSockets and HTTP/2's "h2c" mode.
	//     Body io.ReadCloser

	//     // ContentLength records the length of the associated content. The
	//     // value -1 indicates that the length is unknown. Unless Request.Method
	//     // is "HEAD", values >= 0 indicate that the given number of bytes may
	//     // be read from Body.
	//     ContentLength int64

	//     // Contains transfer encodings from outer-most to inner-most. Value is
	//     // nil, means that "identity" encoding is used.
	//     TransferEncoding []string

	//     // Close records whether the header directed that the connection be
	//     // closed after reading Body. The value is advice for clients: neither
	//     // ReadResponse nor Response.Write ever closes a connection.
	//     Close bool

	//     // Uncompressed reports whether the response was sent compressed but
	//     // was decompressed by the http package. When true, reading from
	//     // Body yields the uncompressed content instead of the compressed
	//     // content actually set from the server, ContentLength is set to -1,
	//     // and the "Content-Length" and "Content-Encoding" fields are deleted
	//     // from the responseHeader. To get the original response from
	//     // the server, set Transport.DisableCompression to true.
	//     Uncompressed bool

	//     // Trailer maps trailer keys to values in the same
	//     // format as Header.
	//     //
	//     // The Trailer initially contains only nil values, one for
	//     // each key specified in the server's "Trailer" header
	//     // value. Those values are not added to Header.
	//     //
	//     // Trailer must not be accessed concurrently with Read calls
	//     // on the Body.
	//     //
	//     // After Body.Read has returned io.EOF, Trailer will contain
	//     // any trailer values sent by the server.
	//     Trailer Header

	//     // Request is the request that was sent to obtain this Response.
	//     // Request's Body is nil (having already been consumed).
	//     // This is only populated for Client requests.
	//     Request *Request

	//     // TLS contains information about the TLS connection on which the
	//     // response was received. It is nil for unencrypted responses.
	//     // The pointer is shared between responses and should not be
	//     // modified.
	//     TLS *tls.ConnectionState
	// }

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
	return response
}
