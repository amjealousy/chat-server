package chat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"wsl.test/shared"
	"wsl.test/utils"
)

type Chat struct {
	ID      string
	mx      sync.RWMutex
	members map[string]*Member
	sseMember
	seq         atomic.Uint64
	messageChan chan shared.TypeMessage
	brakeCtx    context.Context
	brakeCancel context.CancelFunc
	c           *Controller
}

func (c *Chat) RunMessageLoop() bool {
	chatSeq := utils.SeqHistoryChan(c.messageChan)
	go func() {
		chatSeqWithCtx := utils.WithContext(c.brakeCtx, chatSeq)
		for tmsg := range chatSeqWithCtx {
			c.c.logger.InfoContext(c.brakeCtx, "Received message", slog.Any("message", tmsg))
			c.Broadcast(tmsg)

		}
	}()
	return true
}

func NewChat(c *Controller, id string) *Chat {
	chatCtx, cancel := context.WithCancel(context.Background())
	chat := &Chat{
		ID:          id,
		mx:          sync.RWMutex{},
		members:     make(map[string]*Member),
		seq:         atomic.Uint64{},
		messageChan: make(chan shared.TypeMessage, 512),
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
	c.members[member.UserID] = member
	return true
}

func (c *Chat) RemoveMember(userId string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.members[userId] = nil
}
func (c *Chat) RemoveConnection(userId string, conn shared.IConnection) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if member := c.members[userId]; member != nil {
		member.Connection = nil
	}
}

//func (c *Chat) reader(m *Member) {
//	defer c.RemoveConnection(m.ChatID, m.Connection)
//	for {
//		select {
//		case <-m.Connection.GetCTX().Done():
//			return
//		default:
//			select {
//			case msg := <-m.RequestChan:
//				fmt.Println("Chat reach the message from ", m.UserID)
//				c.Broadcast(m, msg)
//
//			}
//		}
//	}
//}

func (c *Chat) Broadcast(msg shared.TypeMessage) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	fmt.Println("Members list ", c.members)
	c.seq.Add(1)

	for _, member := range c.members {
		fmt.Println("broadcast msg", msg)
		if member.UserID == msg.From {
			continue
		}
		select {
		case member.Connection.ReceiveChan() <- msg.Message:

			fmt.Println("Send to Member: ", member.UserID)

		default:
		}
	}

}
