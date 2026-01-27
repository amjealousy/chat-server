package chat

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"wsl.test/shared"
)

type Controller struct {
	addCh  chan Invoice
	delCh  chan Invoice
	mx     *sync.RWMutex
	chats  map[string]*Chat
	logger *slog.Logger
}

func NewController(format io.Writer) *Controller {
	base := slog.New(slog.NewJSONHandler(format, nil))
	logger := base.With(
		slog.String("component", "ChatController"),
	)
	chats := make(map[string]*Chat)
	return &Controller{
		addCh:  make(chan Invoice),
		delCh:  make(chan Invoice),
		chats:  chats,
		logger: logger,
		mx:     &sync.RWMutex{},
	}
}

type Invoice struct {
	ChatID string
	UserID string
}

func (c *Controller) Start(ctx context.Context) {
	c.logger.InfoContext(ctx, "Started")
	defer c.logger.InfoContext(ctx, "Stopped")
	go c.Listener()
	select {
	case <-ctx.Done():
		c.logger.InfoContext(ctx, "Ctx canceled")
	}
}
func (c *Controller) Listener() {
	for {
		select {
		case invoice := <-c.addCh:
			c.logger.InfoContext(context.Background(), "Adding to chat-server", slog.String("chat_id", invoice.ChatID), slog.String("user_id", invoice.UserID))
			c.addToChat(invoice)
		case invoice := <-c.delCh:
			c.delFromChat(invoice)
		default:
		}
	}
}

func (c *Controller) delFromChat(invoice Invoice) {
	if invoice.UserID == "" {
		return
	}
	if chat := c.chats[invoice.ChatID]; chat != nil {
		go chat.RemoveMember(invoice.UserID)
	}
}

func (c *Controller) addToChat(invoice Invoice) {
	if chat := c.chats[invoice.ChatID]; chat != nil {
		c.mx.Lock()
		defer c.mx.Unlock()
		chat.AddMember(invoice.UserID)
	} else {
		c.logger.InfoContext(context.Background(), "The chat-server does not exist :", invoice.ChatID)
	}
}
func (c *Controller) GetChatBroadcastChannel(chatID string, connection shared.IConnection) chan<- shared.TypeMessage {
	c.mx.RLock()
	defer c.mx.RUnlock()
	c.logger.InfoContext(context.Background(), "GetChatBroadcastChannel invoked")
	c.logger.InfoContext(context.Background(), "Existed chats:", slog.Any("chats", c.chats))
	if chat := c.chats[chatID]; chat != nil {
		if member := chat.members[connection.GetUserId()]; member != nil {
			member.AddConnection(connection)
			go chat.reader(member)
			return member.RequestChan
		}
	}
	c.logger.InfoContext(context.Background(), "Chat not found in", slog.String("chatID", chatID))
	return nil
}
func (c *Controller) CreateChat(chatID string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()
	if chat := c.chats[chatID]; chat != nil {
		return false
	}
	c.chats[chatID] = NewChat(chatID)
	return true
}
func (c *Controller) ConnectToChat(invoice Invoice) {
	c.addCh <- invoice
}
