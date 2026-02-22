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

func (s *Server) BindController(controller *chat.Controller) {
	s.controller = controller
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
	server.route.POST("/user/sse", server.SSEChatWrapper)
	server.route.GET("/user/sse", server.SSEPage)
	return server
}

func (s *Server) Run() {
	s.ctx, s.cnl = context.WithCancel(context.Background())
	s.logger.With(
		[]any{
			"port", s.ports[0],
		},
	)
	s.logger.InfoContext(s.ctx, "Listening on port "+s.ports[0])
	err := fasthttp.ListenAndServe(s.ports[0], s.route.Handler)
	s.logger.InfoContext(s.ctx, "Check server error", err)
	if err != nil {

		panic(err)
	}
	select {
	case <-s.ctx.Done():
		return
	}
}

func (s *Server) Stop() {
	s.cnl()
}

func (s *Server) SSEPage(ctx *fasthttp.RequestCtx) {
	htmlContent, err := os.ReadFile("./templates/sse.html")
	if err != nil {
		ctx.Error("404 Not Found", fasthttp.StatusNotFound)
		fmt.Printf("Ошибка чтения файла: %v\n", err)
		return
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.Write(htmlContent)
}

func (s *Server) WSChatWrapper(ctx *fasthttp.RequestCtx) {
	s.logger.InfoContext(s.ctx, "WSChatWrapper called")
	wsHandler(ctx, s.controller)
}
func (s *Server) SSEChatWrapper(ctx *fasthttp.RequestCtx) {
	s.logger.InfoContext(s.ctx, "SSEChatWrapper called")
	sseHandler(ctx, s.controller)
}
func (s *Server) UserCreateWrapper(ctx *fasthttp.RequestCtx) {
	UserCreateHandler(ctx, s.controller)
}

func (s *Server) ChatCreateWrapper(ctx *fasthttp.RequestCtx) {
	ChatCreateHandler(ctx, s.controller)
}
