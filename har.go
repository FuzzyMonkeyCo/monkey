package main

// HAR 1.2 spec handling with nanoseconds & protobuf
// https://github.com/cardigann/harhar
// https://github.com/sebcat/har

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	clientReq    *http.Client
	harCollector *HARRecorder
)

// HARSpecVersion ...
const HARSpecVersion = "1.2"

// HARRecorder logs every HTTP request and response
type HARRecorder struct {
	http.RoundTripper
	Log *HAR_Log
}

// NewHAR is a handy HAR constructor
func NewHAR() *HAR_Log {
	v := time.Now().Format("20060102150405")
	return &HAR_Log{
		Version: HARSpecVersion,
		Creator: &HAR_Creator{
			Version: v,
			Name:    binTitle,
		},
	}
}

// NewRecorder returns a new Recorder object that fulfills the http.RoundTripper interface
func NewRecorder() *HARRecorder {
	return &HARRecorder{
		RoundTripper: http.DefaultTransport,
		Log:          NewHAR(),
	}
}

func newHARTransport() {
	harCollector = NewRecorder()
	clientReq = &http.Client{Transport: harCollector}
}

func isHARReady() bool {
	return clientReq != nil
}

func lastHAR() *HAR_Entry {
	all := harCollector.Log.Entries
	//FIXME: even less data actually needs to be sent
	entry := all[len(all)-1]
	// entry.Request = nil
	return entry
}

func clearHAR() {
	harCollector = nil
	clientReq = nil
}

// Request converts a HAR Request struct to an net/http.Request struct
func (r *HAR_Request) Request() (httpreq *http.Request, err error) {
	dstr := r.PostData.data()
	if len(dstr) > 0 {
		body := strings.NewReader(dstr)
		httpreq, err = http.NewRequest(r.Method, r.URL, body)
	} else {
		httpreq, err = http.NewRequest(r.Method, r.URL, nil)
	}

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(r.Headers); i++ {
		if !strings.Contains(r.Headers[i].Name, ":") {
			httpreq.Header.Add(r.Headers[i].Name, r.Headers[i].Value)
		}
	}

	return httpreq, nil
}

// RoundTrip wraps the HTTP request
func (c *HARRecorder) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ent := &HAR_Entry{}
	if ent.Request, err = harRequest(req); err != nil {
		return nil, err
	}
	// ent.Cache = TODO

	startTime := time.Now()
	if resp, err = c.RoundTripper.RoundTrip(req); err != nil {
		return
	}

	ent.Timings = &HAR_Timings{}
	ent.Timings.Wait = int64(time.Since(startTime))
	ent.Time = ent.Timings.Wait

	ent.Timings.Send = -1    //TODO
	ent.Timings.Receive = -1 //TODO

	ent.StartedDateTime = startTime.Format(time.RFC3339Nano)
	ent.Response, err = harResponse(resp)

	c.Log.Entries = append(c.Log.Entries, ent)
	return
}

// convert an http.Request to a harhar.Request
func harRequest(hr *http.Request) (*HAR_Request, error) {
	r := &HAR_Request{
		Method:      hr.Method,
		URL:         hr.URL.String(),
		HttpVersion: hr.Proto,
		HeadersSize: -1, //TODO
		BodySize:    -1, //TODO
	}

	r.Headers = harHeaders(hr.Header)
	r.Cookies = harCookies(hr.Cookies())

	// parse query params
	qp := hr.URL.Query()
	r.QueryString = make([]*HAR_Query, 0, len(qp))
	for name, vals := range qp {
		for _, val := range vals {
			pair := &HAR_Query{
				Name:  name,
				Value: val,
			}
			r.QueryString = append(r.QueryString, pair)
		}
	}

	if hr.Body == nil {
		return r, nil
	}

	// read in all the data and replace the ReadCloser
	bodyData, err := ioutil.ReadAll(hr.Body)
	if err != nil {
		return r, err
	}
	hr.Body.Close()
	hr.Body = ioutil.NopCloser(bytes.NewReader(bodyData))

	r.PostData.Text = string(bodyData)
	r.PostData.MimeType = hr.Header.Get("Content-Type")
	if r.PostData.MimeType == "" {
		// default per RFC2616
		r.PostData.MimeType = "application/octet-stream"
	}

	return r, nil
}

func harHeaders(headers http.Header) []*HAR_Header {
	hs := make([]*HAR_Header, 0, len(headers))
	for name, vals := range headers {
		for _, val := range vals {
			h := &HAR_Header{
				Name:  name,
				Value: val,
			}
			hs = append(hs, h)
		}
	}
	return hs
}

func harCookies(cookies []*http.Cookie) []*HAR_Cookie {
	nom := make([]*HAR_Cookie, 0, len(cookies))
	for _, c := range cookies {
		nc := &HAR_Cookie{
			Name:     c.Name,
			Path:     c.Path,
			Value:    c.Value,
			Domain:   c.Domain,
			Expires:  c.Expires.Format(time.RFC3339Nano),
			HttpOnly: c.HttpOnly,
			Secure:   c.Secure,
		}
		nom = append(nom, nc)
	}
	return nom
}

// convert an http.Response to a harhar.Response
func harResponse(hr *http.Response) (*HAR_Response, error) {
	r := &HAR_Response{
		Status:      uint32(hr.StatusCode),
		StatusText:  http.StatusText(hr.StatusCode),
		HttpVersion: hr.Proto,
		HeadersSize: -1, //TODO
		BodySize:    -1, //TODO
	}

	r.Headers = harHeaders(hr.Header)
	r.Cookies = harCookies(hr.Cookies())

	// read in all the data and replace the ReadCloser
	bodyData, err := ioutil.ReadAll(hr.Body)
	if err != nil {
		return r, err
	}
	hr.Body.Close()
	hr.Body = ioutil.NopCloser(bytes.NewReader(bodyData))
	r.Content = &HAR_Content{}
	r.Content.Text = string(bodyData)
	r.Content.Size = int32(len(bodyData))

	r.Content.MimeType = hr.Header.Get("Content-Type")
	if r.Content.MimeType == "" {
		// default per RFC2616
		r.Content.MimeType = "application/octet-stream"
	}

	return r, nil
}

// FIXME: URL encoding?
func (p *HAR_PostData) data() string {
	if p == nil {
		return ""
	} else if len(p.Text) > 0 {
		return p.Text
	} else if len(p.Params) > 0 {
		var elems []string
		for _, p := range p.Params {
			var pair string
			if p == nil {
				pair = ""
			} else if len(p.Value) == 0 {
				pair = p.Name
			} else {
				pair = p.Name + "=" + p.Value
			}

			elems = append(elems, pair)
		}

		return strings.Join(elems, "&")
	}

	return ""
}
