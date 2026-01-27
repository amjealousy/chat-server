package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"wsl.test/chat"
	"wsl.test/shared"
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

		u := chat.UserList[user.ID]
		if u == nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid user"))
			return
		} else {
			connection := NewConnection(conn, u)
			connection.Handle(ctx, controller)
		}

	})
}

type Connection struct {
	conn      *websocket.Conn
	Ctx       context.Context
	User      *chat.User
	ID        string
	cachedMap map[string]chan<- shared.TypeMessage
	ReceiveCh chan shared.Message
	cancel    context.CancelFunc
}

func (con *Connection) ReceiveChan() chan shared.Message {
	return con.ReceiveCh
}

func (con *Connection) GetUserId() string {
	return con.User.ID
}

func (con *Connection) GetID() string {
	return con.ID
}

func (con *Connection) GetCTX() context.Context {
	return con.Ctx
}

func (con *Connection) Handle(netCtx *fasthttp.RequestCtx, controller *chat.Controller) {
	fmt.Println("Handling connection")
	defer con.conn.Close()
	go func() {
		for {
			select {
			case message := <-con.ReceiveCh:
				logger.InfoContext(con.Ctx, "Received message", slog.Any("message", message))
				logger.InfoContext(con.Ctx, "Count of message in channel: ", slog.Any("count", len(con.ReceiveCh)))
				con.conn.WriteJSON(message)
			case <-con.Ctx.Done():
				break
			}
		}
	}()
	for {
		select {
		case <-netCtx.Done():
			con.cancel()
			break
		case <-con.Ctx.Done():
			break
		default:
			var message shared.TypeMessage
			_, msgBytes, err := con.conn.ReadMessage()
			if err != nil {
				fmt.Errorf("error after read ws message : %v", err)
			}
			err = json.Unmarshal(msgBytes, &message)
			if err != nil {
				fmt.Errorf("error after parse message : %v", err)
			}
			fmt.Println(message)
			switch message.Type {
			case shared.Broadcast:
				fmt.Println(message)
				if br := con.cachedMap[message.ChatID]; br != nil {
					br <- message
					continue
				}
				invoice := chat.Invoice{ChatID: message.ChatID, UserID: con.User.ID}
				controller.ConnectToChat(invoice)
				if newChannel := controller.GetChatBroadcastChannel(message.ChatID, con); newChannel != nil {
					newChannel <- message
					con.cachedMap[message.ChatID] = newChannel
				}

			}

		}
	}

}

func NewConnection(conn *websocket.Conn, user *chat.User) *Connection {
	id := utils.IDGenerator.Generate()
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		conn:      conn,
		ReceiveCh: make(chan shared.Message, 64),
		ID:        id,
		Ctx:       ctx,
		cancel:    cancel,
		cachedMap: make(map[string]chan<- shared.TypeMessage),
		User:      user,
	}
}
