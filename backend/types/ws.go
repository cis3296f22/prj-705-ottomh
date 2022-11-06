package types

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var timeout = 5 * time.Second // Time between pongs before ws declared dead

// The WebSocket wrapper provides channel-based messag reading
// from the WebSocket, and a convenient Ping / Pong.
type WebSocket struct {
	ws      *websocket.Conn
	r       chan string
	muAlive sync.Mutex
	isAlive bool
}

func (ws *WebSocket) Close() {
	ws.muAlive.Lock()
	ws.isAlive = false
	ws.muAlive.Unlock()
	close(ws.r)
	ws.ws.Close()
}

// Each WebSocket maintains one go thread that just reads messages to address
// an issue in the gorilla websockets API: reads are blocking. So, we
// cannot have the Lobby read from each thread as one inactive WebSocket could
// block the whole program.
// Right now, we assume all messages are test.
func (ws *WebSocket) readCycle() {
	for {
		_, message, err := ws.ws.ReadMessage()
		if err != nil {
			log.Print("Error reading over web socket: ", err)

			// All read errors are permanent and fatal
			ws.Close()

			return
		}

		ws.r <- string(message)
	}
}

// Try to read a message from the web socket. If no message is available,
// it returns an error.
func (ws *WebSocket) ReadMessage() (string, error) {
	select {
	case m := <-ws.r:
		return m, nil
	default:
		return "", errors.New("No message in queue")
	}
}

// Confirms that the WebSocket is still alive.
// Before using a WebSocket, the user should make sure it is alive.
func (ws *WebSocket) IsAlive() bool {
	ws.muAlive.Lock()
	defer ws.muAlive.Unlock()
	return ws.isAlive
}

// Write a message over the web socket.
// TODO: writes can be blocking; how do we handle writes without

func MakeWebSocket(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*WebSocket, error) {
	g_ws, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}

	ws := &WebSocket{
		ws:      g_ws,
		r:       make(chan string, 10),
		muAlive: sync.Mutex{},
		isAlive: true,
	}

	go ws.readCycle()

	return ws, nil
}
