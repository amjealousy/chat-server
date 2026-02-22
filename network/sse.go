package network

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/valyala/fasthttp"
	"wsl.test/chat"
	"wsl.test/internal"
)

func sseHandler(ctx *fasthttp.RequestCtx, controller *chat.Controller) {
	var user chat.User
	if bodyStr := ctx.PostBody(); bodyStr == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	} else {

		if err := json.Unmarshal(bodyStr, &user); err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		if u := chat.GlobalUserList[user.ID]; u == nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.Write([]byte("Invalid user"))
			return
		}
		ctx.Response.Header.Set("Content-Type", "text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		
		ctx.SetStatusCode(fasthttp.StatusOK)
		ticker := time.NewTicker(time.Second)
		wsConnection := SSEConnection{Ctx: ctx, Tick: ticker}
		api := NewAPIConnection(wsConnection, &user)
		api.Handle(ctx, controller)
	}

}

type SSEConnection struct {
	Ctx  *fasthttp.RequestCtx
	Tick *time.Ticker
	a    *APIConnection
	c    *chat.Controller
}

func (S SSEConnection) Processor() func() {
	return func() {
		<-S.Tick.C
	}
}

func (S SSEConnection) Close() error {
	S.Tick.Stop()
	S.Ctx.Response.Header.Set("Connection", "close")
	_, err := S.Ctx.Write([]byte("close"))
	return err
}

func (S SSEConnection) ReadMessage() (messageType int, p []byte, err error) {
	return 0, nil, nil
}

func (S SSEConnection) WriteJSON(msg internal.Message) error {
	if m, err := json.Marshal(msg); err == nil {
		_, werr := S.Ctx.Write(m)
		if werr != nil {
			return errors.Join(err, werr)
		}
		return nil
	} else {
		return err
	}
}
