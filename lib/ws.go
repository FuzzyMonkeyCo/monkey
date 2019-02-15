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
	wsPongWait = 10 * time.Second

	// Maximum message size allowed from peer in bytes
	wsMaxMessageSize = 8 * 1024
)

type wsState struct {
	URL  *url.URL
	c    *websocket.Conn
	req  chan []byte
	rep  chan []byte
	err  chan error
	err2 chan error
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
			ws.err <- err
		}
	}()
	ws.c.SetReadLimit(wsMaxMessageSize)
	ws.c.SetReadDeadline(time.Now().Add(wsPongWait))
	ws.c.SetPingHandler(func(msg string) (err error) {
		log.Println("[DBG] SetPingHandler", msg)
		if err = ws.c.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(wsPongWait)); err != nil {
			log.Println("[ERR]", err)
			ws.err <- err
		}
		return
	})
	for {
		ty, rep, err := ws.c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) { //, websocket.CloseAbnormalClosure) {
				log.Println("[ERR]", err)
			}
			ws.err <- err
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
			ws.err <- err
		}
	}()
	for {
		select {
		case data, ok := <-ws.req:
			ws.c.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				if err := ws.c.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Println("[ERR]", err)
					ws.err <- err
				}
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
				ws.err <- err
				return
			}
		}
	}
}

// NowMonoNano gives current monotonic time with nanoseconds precision
func NowMonoNano() uint64 {
	return uint64(time.Now().UnixNano())
}
