package chat

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"wsl.test/internal"
	"wsl.test/utils"
)

type Chat struct {
	ID          string
	mx          sync.RWMutex
	Members     map[string]*Member
	seq         atomic.Uint64
	MessageChan chan internal.TypeMessage
	brakeCtx    context.Context
	brakeCancel context.CancelFunc
	c           *Controller
}

func (c *Chat) RunMessageLoop() bool {
	chatSeq := utils.SeqHistoryChan(c.MessageChan)
	chatSeqWithCtx := utils.WithContext(c.brakeCtx, chatSeq)
	go func() {
		for tmsg := range chatSeqWithCtx {
			c.c.logger.InfoContext(c.brakeCtx, "Received message", slog.Any("message", tmsg))
			c.Broadcast(tmsg)
			c.c.SendHistory(tmsg)
		}
	}()

	return true
}

func NewChat(c *Controller, id string) *Chat {
	chatCtx, cancel := context.WithCancel(context.Background())
	chat := &Chat{
		ID:          id,
		mx:          sync.RWMutex{},
		Members:     make(map[string]*Member),
		seq:         atomic.Uint64{},
		MessageChan: make(chan internal.TypeMessage, 512),
		brakeCtx:    chatCtx,
		brakeCancel: cancel,
		c:           c,
	}
	chat.RunMessageLoop()
	return chat

}
func (c *Chat) AddMember(user string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()
	member := NewMember(c.ID, user)
	c.Members[member.UserID] = member
	return true
}

func (c *Chat) RemoveMember(userId string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.Members[userId] = nil
}
func (c *Chat) RemoveConnection(userId string, conn internal.IAPIConnection) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if member := c.Members[userId]; member != nil {
		member.Connection = nil
	}
}
func (c *Chat) StopChat() {
	c.brakeCancel()
}

func (c *Chat) Broadcast(msg internal.TypeMessage) {

	c.mx.RLock()
	defer c.mx.RUnlock()
	c.seq.Add(1)
	for _, member := range c.Members {
		if member.UserID == msg.From {
			continue
		}
		select {
		case member.Connection.ReceiveChan() <- msg.Message:

		default:
			c.c.logger.Info("Dropping message", slog.Any("message", msg), slog.String("to", member.UserID))
		}
	}

}
