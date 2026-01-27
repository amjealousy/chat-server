package shared

import (
	"context"
)

type IConnection interface {
	GetID() string
	GetCTX() context.Context
	ReceiveChan() chan Message
	GetUserId() string
}
type Message struct {
	ChatID string `json:"chatID"`
	From   string `json:"from"`
	Cseq   string `json:"cseq"`
	Text   string `json:"text"`
}

type MSGType string

const (
	Broadcast MSGType = "broadcast"
	Mention   MSGType = "@"
)

type TypeMessage struct {
	Message
	Type MSGType `json:"type"`
}
