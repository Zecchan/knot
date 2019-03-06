package knot

import (
	"errors"

	"github.com/gorilla/websocket"
)

type WebSocket struct {
	upgrader *websocket.Upgrader
	Sockets  map[*WebSocketContext]bool
}

func (ws *WebSocket) Upgrade(k *WebContext, fnHandler WebSocketClientHandler) error {
	if ws.upgrader == nil {
		ws.upgrader = &websocket.Upgrader{}
	}
	if ws.Sockets == nil {
		ws.Sockets = map[*WebSocketContext]bool{}
	}

	sck, err := ws.upgrader.Upgrade(k.Writer, k.Request, nil)
	if err != nil {
		return err
	}

	if fnHandler != nil {
		context := WebSocketContext{
			Socket:      sck,
			ExitDesired: false,
			WebContext:  k,
			WebSocket:   ws,
		}
		ws.Sockets[&context] = true
		defer sck.Close()
		defer func() { ws.Sockets[&context] = false }()
		return fnHandler(&context)
	}
	return errors.New("no handler for created websocket, connection will be closed")
}

func (ws *WebSocket) Broadcast(context string, data interface{}, sender *WebSocketContext) {
	bd := WebSocketBroadcastData{
		Context: context,
		Data:    data,
	}
	for clnt := range ws.Sockets {
		if clnt != sender {
			clnt.PushBroadcastData(bd)
		}
	}
}

func (ws *WebSocket) ActiveClients() int {
	if ws.Sockets == nil {
		ws.Sockets = map[*WebSocketContext]bool{}
		return 0
	}
	cnt := 0
	for _, v := range ws.Sockets {
		if v {
			cnt++
		}
	}
	return cnt
}
func (ws *WebSocket) SignalExitToAllClients() {
	if ws.Sockets == nil {
		ws.Sockets = map[*WebSocketContext]bool{}
	}
	for o, v := range ws.Sockets {
		if v {
			o.ExitDesired = true
		}
	}
}

func (ws *WebSocket) IsConnectionClosedError(err error) bool {
	if ce, ok := err.(*websocket.CloseError); ok {
		switch ce.Code {
		case websocket.CloseNormalClosure,
			websocket.CloseGoingAway,
			websocket.CloseNoStatusReceived:
			return true
		}
	}
	return false
}

type WebSocketClientHandler func(context *WebSocketContext) error

type WebSocketBroadcastData struct {
	Context string
	Data    interface{}
}

type WebSocketContext struct {
	ExitDesired   bool
	broadcastData []WebSocketBroadcastData
	Socket        *websocket.Conn
	WebContext    *WebContext
	WebSocket     *WebSocket
}

func (wsc *WebSocketContext) SignalExit() {
	wsc.ExitDesired = true
}

func (wsc *WebSocketContext) PushBroadcastData(data WebSocketBroadcastData) {
	wsc.broadcastData = append(wsc.broadcastData, data)
}

func (wsc *WebSocketContext) PopBroadcastData() (WebSocketBroadcastData, bool) {
	if len(wsc.broadcastData) > 0 {
		tmp := wsc.broadcastData[0]
		wsc.broadcastData = wsc.broadcastData[1:len(wsc.broadcastData)]
		return tmp, true
	}
	return WebSocketBroadcastData{}, false
}
