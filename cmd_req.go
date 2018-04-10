package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/sebcat/har"
)

type harRequest *har.Request

type reqCmd struct {
	V          uint       `json:"v"`
	Cmd        cmdKind    `json:"cmd"`
	Lane       lane       `json:"lane"`
	HARRequest harRequest `json:"har_req"`
}

type reqCmdRep struct {
	V        uint     `json:"v"`
	Cmd      cmdKind  `json:"cmd"`
	Lane     lane     `json:"lane"`
	Us       uint64   `json:"us"`
	HAREntry harEntry `json:"har_rep,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

func (cmd *reqCmd) Kind() cmdKind {
	return cmd.Cmd
}

func (cmd *reqCmd) Exec(cfg *ymlCfg) (rep []byte, err error) {
	lastLane = cmd.Lane
	if !isHARReady() {
		newHARTransport()
	}

	cmd.updateUserAgentHeader()
	if err = cmd.updateURL(cfg); err != nil {
		return
	}
	cmd.updateHostHeader(cfg)
	cmdRep, err := cmd.makeRequest()
	if err != nil {
		return
	}
	totalR++

	if rep, err = json.Marshal(cmdRep); err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func (cmd *reqCmd) makeRequest() (rep *reqCmdRep, err error) {
	r, err := (*cmd.HARRequest).Request()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Printf("[NFO] 🡳\n  ▲  %+v\n", cmd.HARRequest)
	start := time.Now()
	_, err = clientReq.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	log.Printf("[NFO] ❙ %dμs\n", us)
	rep = &reqCmdRep{
		V:    v,
		Cmd:  cmd.Cmd,
		Us:   us,
		Lane: cmd.Lane,
	}

	if err != nil {
		//FIXME: is there a way to describe these failures in HAR 1.2?
		rep.Reason = fmt.Sprintf("%+v", err.Error())
		log.Printf("[NFO]\n  ▼  %s\n", rep.Reason)
		err = nil
		return
	}

	//FIXME maybe: append(headers, fmt.Sprintf("Host: %v", resp.Host))
	//FIXME: make sure order is preserved github.com/golang/go/issues/21853
	rep.HAREntry = lastHAR()
	log.Printf("[NFO]\n  ▼  %+v\n", rep.HAREntry)
	return
}

func (cmd *reqCmd) updateURL(cfg *ymlCfg) (err error) {
	URL, err := url.Parse(cmd.HARRequest.URL)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	// TODO: if host is an IPv6 then it has to be braced with []
	URL.Host = cfg.FinalHost + ":" + cfg.FinalPort
	cmd.HARRequest.URL = URL.String()
	return
}

func (cmd *reqCmd) updateUserAgentHeader() {
	for i := range cmd.HARRequest.Headers {
		if cmd.HARRequest.Headers[i].Name == "User-Agent" {
			if strings.HasPrefix(cmd.HARRequest.Headers[i].Value, "FuzzyMonkey.co/") {
				cmd.HARRequest.Headers[i].Value = binTitle
				break
			}
		}
	}
}

func (cmd *reqCmd) updateHostHeader(cfg *ymlCfg) {
	for i := range cmd.HARRequest.Headers {
		if cmd.HARRequest.Headers[i].Name == "Host" {
			cmd.HARRequest.Headers[i].Value = cfg.FinalHost + ":" + cfg.FinalPort
			break
		}
	}
}
