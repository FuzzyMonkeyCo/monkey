package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:generate go run misc/include_jsons.go

const (
	Version  = "manlion/0.1.0"
	UPSTREAM = "http://localhost:8000/req2.json"
)

func usage() (map[string]interface{}, error) {
	usage := `manlion

Usage:
  manlion test <path> [--slow]
  manlion -h | --help
  manlion --version

Options:
  --slow        Don't phone home using Websockets
  -h --help     Show this screen
  --version     Show version`

	return docopt.Parse(usage, nil, false, Version, false)
}

func rep(req *Req1) (*RepOK1, *RepKO1) {
	var r *http.Request
	var err error
	if req.Payload != nil {
		inPayload := bytes.NewBufferString(*req.Payload)
		r, err = http.NewRequest(req.Method, req.Url, inPayload)
	} else {
		r, err = http.NewRequest(req.Method, req.Url, nil)
	}
	if err != nil {
		log.Fatal("!NewRequest: ", err)
	}
	for _, header := range req.Headers {
		if header == "User-Agent: manlion/1" {
			r.Header.Set("User-Agent", Version)
		} else {
			pair := strings.SplitN(header, ": ", 2)
			r.Header.Set(pair[0], pair[1])
		}
	}
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	if err != nil {
		rep := &RepKO1{
			UID:    req.UID,
			V:      1,
			Us:     us,
			Reason: fmt.Sprintf("%+v", err.Error()),
		}
		return nil, rep
	} else {

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("!read body: ", err)
		}
		var headers []string
		//// headers = append(headers, fmt.Sprintf("Host: %v", resp.Host))
		// Loop through headers
		//FIXME: preserve order github.com/golang/go/issues/21853
		for name, values := range resp.Header {
			name = strings.ToLower(name)
			for _, value := range values {
				headers = append(headers, fmt.Sprintf("%v: %v", name, value))
			}
		}
		rep := &RepOK1{
			UID:     req.UID,
			V:       1,
			Us:      us,
			Code:    resp.StatusCode,
			Headers: headers,
			Payload: string(body),
		}
		return rep, nil
	}
}

func main() {
	args, err := usage()
	if err != nil {
		log.Fatal("!args: ", err)
	}
	//FIXME: use args
	log.Println(args)

	var body []byte
	{
		//FIXME use HTTPS
		resp, err := http.Get(UPSTREAM)
		if err != nil {
			log.Fatal("!fetch: ", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Fatal("!req: ", resp)
		}
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("!ReadAll: ", err)
		}
	}

	{
		bodyString := string(body)
		schemaLoader := gojsonschema.NewStringLoader(REQv1)
		//FIXME: find a loader that works on []byte
		documentLoader := gojsonschema.NewStringLoader(bodyString)
		validation, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			log.Fatal("!Validate: ", err)
		}
		if !validation.Valid() {
			for _, desc := range validation.Errors() {
				log.Printf("\t- %s\n", desc)
			}
			log.Fatal("!validation")
		}
	}

	{
		var req Req1
		if err := json.Unmarshal(body, &req); err != nil {
			log.Fatal("!decode req1: ", err)
		}
		log.Printf("req %+v", req)

		repOK, repKO := rep(&req)
		var rep_json []byte
		if repOK != nil {
			rep_json, err = json.Marshal(repOK)
		} else {
			rep_json, err = json.Marshal(repKO)
		}
		if err != nil {
			log.Fatal("!encode rep1: ", err)
		}
		log.Printf("rep %s", rep_json)

		post, err := http.NewRequest("POST", UPSTREAM, bytes.NewBuffer(rep_json))
		post.Header.Set("Content-Type", "application/json")
		post.Header.Set("Accept", "application/json")
		client := &http.Client{}
		resp, err := client.Do(post)
		if err != nil {
			log.Fatal("!POST: ", err)
		}
		defer resp.Body.Close()
		log.Println("response Status: ", resp.Status)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("response Body: ", string(body))
	}
}
