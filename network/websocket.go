package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"wsl.test/chat"
	"wsl.test/internal"
	"wsl.test/utils"
)

var upgrader = websocket.FastHTTPUpgrader{}

var logger slog.Logger = slog.Logger{}

func wsHandler(ctx *fasthttp.RequestCtx, controller *chat.Controller) {
	if !websocket.FastHTTPIsWebSocketUpgrade(ctx) {
		fmt.Println("WebSocket required")
		ctx.Error("WebSocket required", fasthttp.StatusBadRequest)
		return
	}
	fmt.Println("Upgrader reached")
	upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		defer conn.Close()
		// Читаем ПЕРВОЕ WS-сообщение
		_, bodyBytes, err := conn.ReadMessage()
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Auth read error"))
			return
		}

		var user chat.User
		if err := json.Unmarshal(bodyBytes, &user); err != nil || user.ID == "" {
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid auth JSON"))
			return
		}

		u := chat.GlobalUserList[user.ID]
		if u == nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid user"))
			return
		} else {
			wsConnection := WSConnection{Conn: conn}
			connection := NewAPIConnection(wsConnection, u)
			connection.Handle(ctx, controller)
		}

	})
}

type APIConnection struct {
	conn      internal.Transport
	Ctx       context.Context
	User      *chat.User
	ID        string
	cachedMap map[string]chan<- internal.TypeMessage
	ReceiveCh chan internal.Message
	cancel    context.CancelFunc
	mx        sync.RWMutex
}

func (api *APIConnection) ReceiveChan() chan internal.Message {
	return api.ReceiveCh
}

func (api *APIConnection) GetUserId() string {
	return api.User.ID
}

func (api *APIConnection) GetID() string {
	return api.ID
}

func (api *APIConnection) GetCTX() context.Context {
	return api.Ctx
}

func (api *APIConnection) Handle(netCtx *fasthttp.RequestCtx, controller *chat.Controller) {
	fmt.Println("Handling connection")
	defer api.conn.Close()
	go func() {
		for {
			select {
			case message := <-api.ReceiveCh:
				logger.InfoContext(api.Ctx, "Received message", slog.Any("message", message))
				logger.InfoContext(api.Ctx, "Count of message in channel: ", slog.Any("count", len(api.ReceiveCh)))
				api.conn.WriteJSON(message)
			case <-api.Ctx.Done():
				break
			}
		}
	}()
	for {
		select {
		case <-netCtx.Done():
			api.cancel()
			break
		default:
			handler := api.conn.Processor()
			handler()

		}
	}

}

func (api *APIConnection) Register(c *chat.Controller, invoice chat.Invoice, message internal.TypeMessage) {
	c.ConnectToChat(invoice)
	eventC := make(chan chat.Event)
	err := c.Subscribe(api.User.ID, eventC)
	if err != nil {
		return
	} else {
		select {
		case event := <-eventC:
			switch event.Type {
			case chat.Completed:
				if chatChan := c.GetRegisterConnection(invoice.ChatID, api); chatChan != nil {
					chatChan <- message
					api.mx.Lock()
					api.cachedMap[invoice.ChatID] = chatChan
					api.mx.Unlock()
				}
			case chat.Failed:
				return
			}
		}
	}

}

func NewAPIConnection(conn internal.Transport, user *chat.User) *APIConnection {
	id := utils.IDGenerator.Generate()
	ctx, cancel := context.WithCancel(context.Background())
	return &APIConnection{
		conn:      conn,
		ReceiveCh: make(chan internal.Message, 64),
		ID:        id,
		Ctx:       ctx,
		cancel:    cancel,
		cachedMap: make(map[string]chan<- internal.TypeMessage),
		User:      user,
		mx:        sync.RWMutex{},
	}
}

type WSConnection struct {
	Conn *websocket.Conn
	a    *APIConnection
	c    *chat.Controller
}

func (W WSConnection) Close() error {
	return W.Conn.Close()
}

func (W WSConnection) ReadMessage() (messageType int, p []byte, err error) {
	return W.Conn.ReadMessage()
}

func (W WSConnection) WriteJSON(msg internal.Message) error {
	return W.Conn.WriteJSON(msg)
}
func (W WSConnection) Processor() func() {
	return func() {

		var message internal.TypeMessage
		_, msgBytes, err := W.ReadMessage()
		if err != nil {
			fmt.Errorf("error after read ws message : %v", err)
		}
		err = json.Unmarshal(msgBytes, &message)
		if err != nil {
			fmt.Errorf("error after parse message : %v", err)
		}
		fmt.Println(message)
		switch message.Type {
		case internal.Broadcast:
			fmt.Println(message)
			W.a.mx.RLock()
			if br := W.a.cachedMap[message.ChatID]; br != nil {
				br <- message
			} else {
				invoice := chat.Invoice{ChatID: message.ChatID, UserID: W.a.User.ID}
				go W.a.Register(W.c, invoice, message)
			}
			W.a.mx.RUnlock()

		}
	}

}
