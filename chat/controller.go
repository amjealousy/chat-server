package chat

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"wsl.test/internal"
	"wsl.test/processor"
)

type Controller struct {
	addCh  chan Invoice
	delCh  chan Invoice
	mx     *sync.RWMutex
	Chats  map[string]*Chat
	logger *slog.Logger
	sentry *Sentry
	kpool  *processor.ProducerPool
}
type EventType = string

const (
	Completed EventType = "complete"
	Failed    EventType = "failed"
)

type Event struct {
	Type EventType `json:"type"`
	Err  string    `json:"err"`
}

type Sentry struct {
	mx        *sync.RWMutex
	notifyMap map[string]chan Event
}

func NewSentry() *Sentry {
	return &Sentry{&sync.RWMutex{}, make(map[string]chan Event)}
}
func (s *Sentry) Notify(userID string, ev Event) {
	s.mx.RLock()
	userC := s.notifyMap[userID]
	s.mx.RUnlock()
	select {
	case userC <- ev:
	default:
	}
}

func (c *Controller) Subscribe(userId string, ch chan Event) error {
	if _, ok := c.sentry.notifyMap[userId]; ok {
		return errors.New("already subscribed")
	}
	if ch == nil {
		return errors.New("nil channel")
	}
	c.sentry.mx.Lock()
	c.sentry.notifyMap[userId] = ch
	c.sentry.mx.Unlock()
	return nil
}

func NewController(format io.Writer, pool *processor.ProducerPool) *Controller {
	base := slog.New(slog.NewJSONHandler(format, nil))
	logger := base.With(
		slog.String("O", "[ChatController]"),
	)
	chats := make(map[string]*Chat)
	return &Controller{
		addCh:  make(chan Invoice),
		delCh:  make(chan Invoice),
		Chats:  chats,
		logger: logger,
		mx:     &sync.RWMutex{},
		kpool:  pool,
		sentry: NewSentry(),
	}
}

type Invoice struct {
	ChatID string
	UserID string
}

func (c *Controller) Start(ctx context.Context) {
	c.logger.InfoContext(ctx, "Started")
	defer c.logger.InfoContext(ctx, "Stopped")
	go c.Listener(ctx)
	select {
	case <-ctx.Done():
		c.logger.InfoContext(ctx, "Ctx canceled")
	}
}
func (c *Controller) Listener(ctx context.Context) {
	for {
		select {
		case invoice := <-c.addCh:
			c.logger.InfoContext(context.Background(), "Adding to chat-server", slog.String("chat_id", invoice.ChatID), slog.String("user_id", invoice.UserID))
			c.addToChat(invoice)
		case invoice := <-c.delCh:
			c.delFromChat(invoice)
		case <-ctx.Done():
			break
		default:
		}
	}
}

func (c *Controller) delFromChat(invoice Invoice) {
	if invoice.UserID == "" {
		return
	}
	if chat := c.Chats[invoice.ChatID]; chat != nil {
		go chat.RemoveMember(invoice.UserID)
	}
}

func (c *Controller) addToChat(invoice Invoice) {
	var event Event
	if chat := c.Chats[invoice.ChatID]; chat != nil {
		c.mx.Lock()
		defer c.mx.Unlock()
		status := chat.AddMember(invoice.UserID)
		if status == false {
			event.Type = Failed
			event.Err = "failed to add to chat"
		} else {
			event.Type = Completed
		}

	} else {
		c.logger.InfoContext(context.Background(), "The chat-server does not exist :", invoice.ChatID)
		event.Type = Failed
		event.Err = "chat-server does not exist"
	}
	c.sentry.Notify(invoice.UserID, event)
}
func (c *Controller) GetRegisterConnection(chatID string, connection internal.IAPIConnection) chan internal.TypeMessage {
	c.mx.RLock()
	defer c.mx.RUnlock()
	c.logger.InfoContext(context.Background(), "[Controller.GetRegisterConnection] list of chats:", slog.Any("chats", c.Chats))
	if chat := c.Chats[chatID]; chat != nil {
		if member := chat.Members[connection.GetUserId()]; member != nil {
			return chat.MessageChan
		}
	}
	c.logger.InfoContext(context.Background(), "Chat not found in", slog.String("chatID", chatID))
	return nil
}
func (c *Controller) CreateChat(chatID string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()
	if chat := c.Chats[chatID]; chat != nil {
		return false
	}
	c.Chats[chatID] = NewChat(c, chatID)
	return true
}
func (c *Controller) ConnectToChat(invoice Invoice) {
	c.addCh <- invoice
}
func (c *Controller) SseConnect(invoice Invoice) {}

type CallBack func()

func (c *Controller) SendHistory(message internal.TypeMessage) {
	c.kpool.Post(ConvertHistoryMessage(message))
}
func ConvertHistoryMessage(message internal.TypeMessage) processor.HistoryMessage {
	return processor.HistoryMessage{
		Seq:    message.Cseq,
		UserID: message.From,
		ChatID: message.ChatID,
		Text:   message.Text,
		Ts:     time.Now().Unix(),
	}
}
