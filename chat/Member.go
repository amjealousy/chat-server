package chat

import (
	"wsl.test/internal"
)

type Member struct {
	ChatID      string
	UserID      string
	Connection  internal.IAPIConnection
	RequestChan chan internal.TypeMessage
}

func NewMember(ChatID string, UserID string) *Member {
	return &Member{
		ChatID:      ChatID,
		UserID:      UserID,
		RequestChan: make(chan internal.TypeMessage),
	}
}
func (m *Member) AddConnection(conn internal.IAPIConnection) {
	m.Connection = conn
}
func (m *Member) RemoveConnection() {
	m.Connection = nil
}
