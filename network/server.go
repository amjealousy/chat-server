package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"wsl.test/chat"
)

func HealthHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(200)
	ctx.SetContentType("text/plain; charset=utf-8")
	ctx.SetBodyString(`{"alive": true}`)
}
func ChatCreateHandler(ctx *fasthttp.RequestCtx, controller *chat.Controller) {
	ctx.SetContentType("application/json")
	chatID := ctx.UserValue("chatID").(string)
	fmt.Printf("Requested id: %s\n", chatID)
	if status := controller.CreateChat(chatID); status {
		ctx.SetStatusCode(fasthttp.StatusCreated)
	} else {
		ctx.SetStatusCode(fasthttp.StatusConflict)
	}
}

type UserRequest struct {
	FQDN string `json:"fqdn"`
}

func UserCreateHandler(ctx *fasthttp.RequestCtx, controller *chat.Controller) {
	if body := ctx.PostBody(); body == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		fmt.Fprintf(ctx, "body is empty!")
		return
	} else {
		var ur UserRequest
		if err := json.Unmarshal(body, &ur); err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
		}
		user := chat.CreateUser(ur.FQDN)
		ctx.SetStatusCode(fasthttp.StatusCreated)
		fmt.Fprintf(ctx, "User: %s", user.ID)
	}
}

type Server struct {
	route      *router.Router
	controller *chat.Controller
	ports      []string
	ctx        context.Context
	cnl        context.CancelFunc
	logger     *slog.Logger
}

func (server *Server) BindController(controller *chat.Controller) {
	server.controller = controller

}
func NewServer() *Server {
	server := new(Server)
	server.route = router.New()
	server.ports = []string{":5099"}
	base := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server.logger = base.With(
		slog.String("component", "HttpServer"),
	)
	server.route.GET("/health", HealthHandler)
	server.route.GET("/ws", server.WSChatWrapper)
	server.route.POST("/chat-server/create/{chatID}", server.ChatCreateWrapper)
	server.route.POST("/user/create", server.UserCreateWrapper)
	return server
}

func (server *Server) Run() {
	server.ctx, server.cnl = context.WithCancel(context.Background())
	server.logger.With(
		[]any{
			"port", server.ports[0],
		},
	)
	server.logger.InfoContext(server.ctx, "Listening on port "+server.ports[0])
	err := fasthttp.ListenAndServe(server.ports[0], server.route.Handler)
	server.logger.InfoContext(server.ctx, "Check server error", err)
	if err != nil {

		panic(err)
	}
	select {
	case <-server.ctx.Done():
		return
	}
}

func (server *Server) Stop() {
	server.cnl()
}

func (server *Server) WSChatWrapper(ctx *fasthttp.RequestCtx) {
	server.logger.InfoContext(server.ctx, "WSChatWrapper called")
	wsHandler(ctx, server.controller)
}

func (server *Server) UserCreateWrapper(ctx *fasthttp.RequestCtx) {
	UserCreateHandler(ctx, server.controller)
}

func (server *Server) ChatCreateWrapper(ctx *fasthttp.RequestCtx) {
	ChatCreateHandler(ctx, server.controller)
}
