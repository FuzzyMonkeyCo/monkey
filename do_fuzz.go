package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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

func newFuzz(cfg *UserCfg, vald *validator) (cmd someCmd, err error) {
	initer := &MsgFuzz{
		Cfg:  cfg,
		Spec: vald.Spec,
	}
	payload, err := proto.Marshal(initer)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if err = maybePreStart(cfg); err != nil {
		return
	}

	cmdJSON, err := fuzzNew(cfg, payload)
	if err != nil {
		return
	}

	cmd, err = unmarshalCmd(cmdJSON)
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

func fuzzNew(cfg *UserCfg, payload []byte) (rep []byte, err error) {

	doDaDING() ///FIXME

	r, err := http.NewRequest(http.MethodPut, apiFuzzNew, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	r.Header.Set(headerContentType, mimeYAML)
	r.Header.Set(headerAccept, mimeJSON)
	r.Header.Set(headerUserAgent, binTitle)
	r.Header.Set(headerXAuthToken, cfg.AuthToken)

	log.Printf("[DBG] 🡱  PUT %s\n  🡱  %dB\n", apiFuzzNew, len(payload))
	start := time.Now()
	resp, err := clientUtils.Do(r)
	log.Printf("[DBG] ❙ %dμs\n", time.Since(start)/time.Microsecond)
	if err != nil {
		// if here probably a HomeConnectionError
		log.Println("[ERR]", err)
		return
	}
	defer resp.Body.Close()

	if rep, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG]\n  🡳  %s\n", rep)

	if resp.StatusCode != http.StatusCreated {
		err = newStatusError(http.StatusCreated, resp.Status)
		log.Println("[ERR]", err)
		return
	}

	cfg.AuthToken = resp.Header.Get(headerXAuthToken)
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

///////////

func doDaDING() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "localhost:1042", Path: "/fuzz"}
	log.Printf("connecting to %s", u.String())

	headers := http.Header{headerUserAgent: {binTitle}, headerXAuthToken: {"bla"}}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			if err := c.WriteMessage(websocket.TextMessage, []byte(t.String())); err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(1 * time.Second):
			}
			return
		}
	}
}
