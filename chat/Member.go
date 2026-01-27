package chat

import (
	"wsl.test/shared"
)

type Member struct {
	ChatID      string
	UserID      string
	Connection  shared.IConnection
	RequestChan chan shared.TypeMessage
}

func NewMember(ChatID string, UserID string) *Member {
	return &Member{
		ChatID:      ChatID,
		UserID:      UserID,
		RequestChan: make(chan shared.TypeMessage),
	}
}
func (m *Member) AddConnection(conn shared.IConnection) {
	m.Connection = conn
}
