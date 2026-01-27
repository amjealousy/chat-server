package chat

import (
	"fmt"
	"sync"
	"sync/atomic"

	"wsl.test/shared"
)

type Chat struct {
	ID      string
	mx      sync.RWMutex
	members map[string]*Member
	seq     atomic.Uint64
}

func NewChat(id string) *Chat {
	a := atomic.Uint64{}
	a.Store(0)
	return &Chat{
		ID:      id,
		mx:      sync.RWMutex{},
		members: make(map[string]*Member),
		seq:     a,
	}
}
func (c *Chat) AddMember(user string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	member := NewMember(c.ID, user)
	c.members[member.UserID] = member
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

func (c *Chat) reader(m *Member) {
	defer c.RemoveConnection(m.ChatID, m.Connection)
	for {
		select {
		case <-m.Connection.GetCTX().Done():
			return
		default:
			select {
			case msg := <-m.RequestChan:
				fmt.Println("Chat reach the message from ", m.UserID)
				c.Broadcast(m, msg)

			}
		}
	}
}
func (c *Chat) Broadcast(m *Member, msg shared.TypeMessage) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	fmt.Println("Members list ", c.members)
	add := c.seq.Add(1)
	for _, member := range c.members {
		fmt.Println("broadcast msg", msg)
		if member.UserID == m.UserID {
			continue
		}
		select {
		case member.Connection.ReceiveChan() <- msg.Message:
			fmt.Println("Send to Member: ", member.UserID)

			// todo logic to serve in chat-history
		default:
		}
	}
}
