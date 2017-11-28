package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type reqCmd struct {
	V       uint     `json:"v"`
	Cmd     string   `json:"cmd"`
	Lane    lane     `json:"lane"`
	Method  string   `json:"method"`
	URL     string   `json:"url"`
	Headers []string `json:"headers"`
	Payload *string  `json:"payload"`
}

type reqCmdRep struct {
	V      uint   `json:"v"`
	Cmd    string `json:"cmd"`
	Lane   lane   `json:"lane"`
	Us     uint64 `json:"us"`
	HAR    har    `json:"har,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (cmd *reqCmd) Kind() string {
	return cmd.Cmd
}

func (cmd *reqCmd) Exec(cfg *ymlCfg) (rep []byte, err error) {
	lastLane = cmd.Lane

	cmdURL, err := updateURL(cfg, cmd.URL)
	if err != nil {
		return
	}
	cmdRep, err := cmd.makeRequest(cmdURL)
	if err != nil {
		return
	}

	rep, err = json.Marshal(cmdRep)
	if err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func updateURL(cfg *ymlCfg, URL string) (updatedURL string, err error) {
	u, err := url.Parse(URL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	// Note: if host is an IPv6 then it has to be braced with []
	u.Host = cfg.FinalHost + ":" + cfg.FinalPort
	updatedURL = u.String()
	return
}

func (cmd *reqCmd) makeRequest(url string) (rep *reqCmdRep, err error) {
	var r *http.Request
	var _pld string
	if cmd.Payload != nil {
		_pld = *cmd.Payload
		inPayload := bytes.NewBufferString(*cmd.Payload)
		r, err = http.NewRequest(cmd.Method, url, inPayload)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
	} else {
		_pld = ""
		r, err = http.NewRequest(cmd.Method, url, nil)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
	}

	if !isHARReady() {
		newHARTransport()
	}

	for _, header := range cmd.Headers {
		if header == "User-Agent: CoveredCI-passthrough/1" {
			r.Header.Set("User-Agent", binTitle)
		} else {
			pair := strings.SplitN(header, ": ", 2)
			r.Header.Set(pair[0], pair[1])
		}
	}

	start := time.Now()
	_, err = clientReq.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)

	rep = &reqCmdRep{
		V:    1,
		Cmd:  cmd.Cmd,
		Lane: cmd.Lane,
		Us:   us,
	}

	if err != nil {
		reason := fmt.Sprintf("%+v", err.Error())
		log.Printf("[NFO] 🡳  %vμs %s %s\n  ▲  %s\n  ▼  %s\n", us, cmd.Method, url, _pld, reason)
		rep.Reason = reason
		return
	}

	//FIXME maybe: append(headers, fmt.Sprintf("Host: %v", resp.Host))
	//FIXME: make sure order is preserved github.com/golang/go/issues/21853
	rep.HAR = lastHAR()
	return
}
