package lib

import (
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Fail if sending takes longer
	wsWriteWait = 2 * time.Second

	// Time allowed to read the next pong message from the peer.
	wsPongWait = 30 * time.Second

	// Maximum message size allowed from peer in bytes
	wsMaxMessageSize = 8192

	// Fail if reading for longer
	wsTimeout = 15 * time.Second
)

type wsState struct {
	URL    *url.URL
	c      *websocket.Conn
	msgUID uint32
	req    chan []byte
	rep    chan []byte
	err    chan error
	err2   chan error
}

func newWS(URL *url.URL, c *websocket.Conn) *wsState {
	ws := &wsState{
		URL:  URL,
		c:    c,
		req:  make(chan []byte, 1),
		rep:  make(chan []byte, 1),
		err:  make(chan error, 1),
		err2: make(chan error, 1),
	}
	go ws.reader()
	go ws.writer()
	return ws
}

func (ws *wsState) reader() {
	log.Println("[DBG] starting reader")
	defer func() {
		log.Println("[DBG] ending reader")
		if err := ws.c.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}()
	ws.c.SetReadLimit(wsMaxMessageSize)
	ws.c.SetReadDeadline(time.Now().Add(wsPongWait))
	ws.c.SetPingHandler(func(FIXME string) error {
		log.Println("[DBG] SetPingHandler", FIXME)
		ws.c.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})
	for {
		ty, rep, err := ws.c.ReadMessage()
		if err != nil {
			ws.err <- err
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) { //, websocket.CloseAbnormalClosure) {
				log.Println("[ERR]", err)
			}
			break
		}
		log.Printf("[DBG] recv %dB: %s... (%#v)\n", len(rep), rep[:4], ty)
		ws.rep <- rep
	}
}

func (ws *wsState) writer() {
	log.Println("[DBG] starting writer")
	defer func() {
		log.Println("[DBG] ending writer")
		if err := ws.c.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}()
	for {
		select {
		case data, ok := <-ws.req:
			ws.c.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				ws.c.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			log.Println("[DBG] <-req", data[:4])
			start := time.Now()
			if err := ws.c.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Println("[ERR]", err)
				ws.err <- err
				return
			}
			log.Println("[DBG] sent", len(data), "in", time.Since(start))

		case err := <-ws.err2:
			log.Println("[DBG] <-err2")
			data := []byte(err.Error())
			if err := ws.c.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("[ERR]", err)
				return
			}

		case <-time.After(wsTimeout):
			ColorERR.Println(ws.URL.Hostname(), "took too long to respond")
			log.Fatalln("[ERR] srvTimeout")
			return
		}
	}
}
