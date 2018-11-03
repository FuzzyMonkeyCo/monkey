package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	v                 = 1
	mimeJSON          = "application/json"
	mimeYAML          = "application/x-yaml"
	headerContentType = "Content-Type"
	headerAccept      = "Accept"
	headerUserAgent   = "User-Agent"
	headerXAPIKey     = "X-Api-Key"
	headerXAuthToken  = "X-Auth-Token"
)

var ws *wsState
var wsMsgUID uint64

type wsState struct {
	req  chan []byte
	rep  chan []byte
	err  chan error
	ping chan struct{}
	done chan struct{}
}

func (ws *wsState) call(req *Msg) (rep *Msg, err error) {
	// FIXME: take a Writer, return a Reader

	// NOTE: log in caller
	wsMsgUID++
	req.UID = wsMsgUID
	payload, err := proto.Marshal(req)
	if err != nil {
		return
	}

	log.Printf("[DBG] 🡱 PUT %dB\n", len(payload))
	start := time.Now()
	ws.req <- payload
	select {
	case payload = <-ws.rep:
		log.Println("[DBG] 🡳", time.Now().Sub(start), payload[:4])
		var msg Msg
		if err = proto.Unmarshal(payload, &msg); err != nil {
			return
		}

		if msg.GetUID() != req.UID {
			err = errors.New("bad dialog sequence number")
			return
		}

		switch msg.GetMsg().(type) {
		case *Msg_Err500:
			err = errors.New("500 Internal Server Error")
		case *Msg_Err400:
			err = errors.New("400 Bad Request")
		case *Msg_Err401:
			err = errors.New("401 Unauthorized")
		case *Msg_Err403:
			err = errors.New("403 Forbidden")
		default:
			rep = &msg
		}
		return
	case err = <-ws.err:
		log.Println("[DBG] 🡳", time.Now().Sub(start), err)
		return
	case <-time.After(15 * time.Second):
		err = errors.New("ws call timeout")
		return
	}
}

func newFuzz(cfg *UserCfg, vald *validator) (cmd someCmd, err error) {
	if err = newWS(cfg); err != nil {
		return
	}

	msg := &Msg{Msg: &Msg_Fuzz{Fuzz: &ReqFuzz{
		Cfg:  cfg,
		Spec: vald.Spec,
	}}}

	if msg, err = ws.call(msg); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	switch msg.GetMsg().(type) {
	case *Msg_CmdReset:
		cmd = msg.GetCmdReset()
		return
	case *Msg_CmdStart:
		cmd = msg.GetCmdReset()
		return
	default:
		err = errors.New("unexpected msg")
		log.Println("[ERR]", err)
		return
	}
}

func (cmd *RepCmdReset) isMsg_Msg() {}

func (cmd *RepCmdReset) Kind() cmdKind {
	return kindReset
}

func (cmd *RepCmdReset) Exec(cfg *UserCfg) (rep []byte, err error) {
	return
}

func fuzzNext(cfg *UserCfg, cmd someCmd) (someCmd someCmd, err error) {
	// Sometimes sets cfg.Runtime.Final* fields
	rep, err := cmd.Exec(cfg)
	if err != nil {
		return
	}

	nextCmdJSON, err := nextPOST(cfg, rep)
	if err != nil {
		return
	}

	someCmd, err = unmarshalCmd(nextCmdJSON)
	return
}

func nextPOST(cfg *UserCfg, payload []byte) (rep []byte, err error) {
	r, err := http.NewRequest(http.MethodPost, apiFuzzNext, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set(headerContentType, mimeJSON)
	r.Header.Set(headerAccept, mimeJSON)
	r.Header.Set(headerUserAgent, binTitle)
	r.Header.Set(headerXAuthToken, cfg.AuthToken)

	log.Printf("[DBG] 🡱  POST %s\n  🡱  %s\n", apiFuzzNext, payload)
	start := time.Now()
	resp, err := clientUtils.Do(r)
	log.Printf("[DBG] ❙ %dμs\n", time.Since(start)/time.Microsecond)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if rep, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG]\n  🡳  %s\n", rep)

	if resp.StatusCode != http.StatusOK {
		err = newStatusError(http.StatusOK, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	cfg.AuthToken = resp.Header.Get(headerXAuthToken)
	return
}

func newWS(cfg *UserCfg) error {
	headers := http.Header{
		headerUserAgent:  {binTitle},
		headerXAuthToken: {cfg.AuthToken},
	}
	//FIXME
	u := &url.URL{Scheme: "ws", Host: "localhost:1042", Path: "/1/fuzz"}

	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}

	ws = &wsState{
		req:  make(chan []byte, 1),
		rep:  make(chan []byte, 1),
		err:  make(chan error, 1),
		ping: make(chan struct{}, 1),
		done: make(chan struct{}, 1),
	}

	go func() {
		defer func() { ws.done <- struct{}{} }()
		for {
			_, rep, err := c.ReadMessage()
			if err != nil {
				log.Println("[ERR]", err)
				ws.err <- err
				return
			}
			log.Printf("recv %dB: %s...\n", len(rep), rep[:4])
			if len(rep) == 4 && string(rep) == `PING` {
				ws.ping <- struct{}{}
			} else {
				ws.rep <- rep
			}
		}
		log.Println("ws reader ending")
	}()

	var once sync.Once
	pong := make(chan struct{}, 1)
	srvTimeout := 15 * time.Second

	go func() {
		defer close(ws.req)
		defer close(ws.rep)
		defer close(ws.err)
		defer close(ws.ping)
		defer close(ws.done)

		func() {
			for {
				select {
				case data := <-ws.req:
					log.Println("ws.req")
					if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
						log.Println("[ERR]", err)
						ws.err <- err
						return
					}
				case <-ws.ping:
					log.Println("ws.ping")
					data := []byte(`PONG`)
					if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Println("[ERR]", err)
						ws.err <- err
						return
					}
					once.Do(func() { pong <- struct{}{} })
				case <-ws.done:
					log.Println("ws.done")
					return
				case <-time.After(srvTimeout):
					log.Fatalln("[ERR] srvTimeout")
					return
				}
			}
		}()
		log.Println("ws manager ending")
		if err := c.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}()

	select {
	case <-pong:
		log.Println("pong!")
		close(pong)
		return nil
	case err := <-ws.err:
		log.Println("err!")
		return err
	case <-time.After(srvTimeout):
		err := errors.New("timeout waiting for PING from server")
		log.Println("[ERR]", err)
		return err
	}
}
