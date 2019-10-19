package modeler_openapiv3

// // HAR 1.2 spec handling with nanoseconds & protobuf
// // https://github.com/cardigann/harhar
// // https://github.com/sebcat/har

// import (
// 	"bytes"
// 	"io/ioutil"
// 	"net/http"
// 	"strings"
// 	"time"
// )

// func harEntry(req *http.Request, rep *http.Response, elapsed time.Duration) (ent *HAR_Entry, err error) {
// 	ent = &HAR_Entry{}
// 	if ent.Request, err = harRequest(req); err != nil {
// 		return nil, err
// 	}
// 	// ent.Cache = TODO

// 	ent.Timings = &HAR_Timings{}
// 	ent.Timings.Wait = int64(elapsed)
// 	ent.Time = ent.Timings.Wait

// 	ent.Timings.Send = -1    //TODO
// 	ent.Timings.Receive = -1 //TODO

// 	// ent.StartedDateTime = startTime.Format(time.RFC3339Nano)
// 	ent.Response, err = harResponse(rep)

// 	return
// }

// // convert an http.Request to a harhar.Request
// func harRequest(hr *http.Request) (*HAR_Request, error) {
// 	r := &HAR_Request{
// 		Method:      hr.Method,
// 		URL:         hr.URL.String(),
// 		HttpVersion: hr.Proto,
// 		HeadersSize: -1, //TODO
// 		BodySize:    -1, //TODO
// 	}

// 	r.Headers = harHeaders(hr.Header)
// 	r.Cookies = harCookies(hr.Cookies())

// 	// parse query params
// 	qp := hr.URL.Query()
// 	r.QueryString = make([]*HAR_Query, 0, len(qp))
// 	for name, vals := range qp {
// 		for _, val := range vals {
// 			pair := &HAR_Query{
// 				Name:  name,
// 				Value: val,
// 			}
// 			r.QueryString = append(r.QueryString, pair)
// 		}
// 	}

// 	if hr.Body == nil {
// 		return r, nil
// 	}

// 	// read in all the data and replace the ReadCloser
// 	bodyData, err := ioutil.ReadAll(hr.Body)
// 	if err != nil {
// 		return r, err
// 	}
// 	hr.Body.Close()
// 	hr.Body = ioutil.NopCloser(bytes.NewReader(bodyData))

// 	r.PostData.Text = string(bodyData)
// 	r.PostData.MimeType = hr.Header.Get("Content-Type")
// 	if r.PostData.MimeType == "" {
// 		// default per RFC2616
// 		r.PostData.MimeType = "application/octet-stream"
// 	}

// 	return r, nil
// }

// func harHeaders(headers http.Header) []*HAR_Header {
// 	hs := make([]*HAR_Header, 0, len(headers))
// 	for name, vals := range headers {
// 		for _, val := range vals {
// 			h := &HAR_Header{
// 				Name:  name,
// 				Value: val,
// 			}
// 			hs = append(hs, h)
// 		}
// 	}
// 	return hs
// }

// func harCookies(cookies []*http.Cookie) []*HAR_Cookie {
// 	nom := make([]*HAR_Cookie, 0, len(cookies))
// 	for _, c := range cookies {
// 		nc := &HAR_Cookie{
// 			Name:     c.Name,
// 			Path:     c.Path,
// 			Value:    c.Value,
// 			Domain:   c.Domain,
// 			Expires:  c.Expires.Format(time.RFC3339Nano),
// 			HttpOnly: c.HttpOnly,
// 			Secure:   c.Secure,
// 		}
// 		nom = append(nom, nc)
// 	}
// 	return nom
// }

// // convert an http.Response to a harhar.Response
// func harResponse(hr *http.Response) (*HAR_Response, error) {
// 	r := &HAR_Response{
// 		Status:      uint32(hr.StatusCode),
// 		StatusText:  http.StatusText(hr.StatusCode),
// 		HttpVersion: hr.Proto,
// 		HeadersSize: -1, //TODO
// 		BodySize:    -1, //TODO
// 	}

// 	r.Headers = harHeaders(hr.Header)
// 	r.Cookies = harCookies(hr.Cookies())

// 	// read in all the data and replace the ReadCloser
// 	bodyData, err := ioutil.ReadAll(hr.Body)
// 	if err != nil {
// 		return r, err
// 	}
// 	hr.Body.Close()
// 	hr.Body = ioutil.NopCloser(bytes.NewReader(bodyData))
// 	r.Content = &HAR_Content{}
// 	r.Content.Text = string(bodyData)
// 	r.Content.Length = int32(len(bodyData))

// 	r.Content.MimeType = hr.Header.Get("Content-Type")
// 	if r.Content.MimeType == "" {
// 		// default per RFC2616
// 		r.Content.MimeType = "application/octet-stream"
// 	}

// 	return r, nil
// }

// // FIXME: URL encoding?
// func (p *HAR_PostData) data() string {
// 	switch {
// 	case p == nil:
// 		return ""
// 	case len(p.Text) > 0:
// 		return p.Text
// 	case len(p.Params) > 0:
// 		var elems []string
// 		for _, p := range p.Params {
// 			var pair string
// 			switch {
// 			case p == nil:
// 				pair = ""
// 			case len(p.Value) == 0:
// 				pair = p.Name
// 			default:
// 				pair = p.Name + "=" + p.Value
// 			}

// 			elems = append(elems, pair)
// 		}

// 		return strings.Join(elems, "&")
// 	}

// 	return ""
// }
