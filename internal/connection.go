package internal

import (
	"context"
)

type IAPIConnection interface {
	GetID() string
	GetCTX() context.Context
	ReceiveChan() chan Message
	GetUserId() string
}
type Transport interface {
	Close() error
	ReadMessage() (messageType int, p []byte, err error)
	WriteJSON(msg Message) error
	Processor() func()
}
