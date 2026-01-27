package chat

import (
	"context"
	"fmt"

	"wsl.test/shared"
)

type Observer struct {
	Id        string
	ReceiveCh chan shared.TypeMessage
	Ctx       context.Context
	kickFunc  context.CancelFunc
}

func NewObserver(id string) *Observer {
	ch := make(chan shared.TypeMessage)
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &Observer{
		Id:        id,
		ReceiveCh: ch,
		Ctx:       ctx,
		kickFunc:  cancelFunc,
	}
}
func (ob *Observer) Send(message shared.TypeMessage) {
	select {
	case ob.ReceiveCh <- message:
	default:
		fmt.Println("[Observer] Channel full")
	}
}
func (ob *Observer) Kick() {
	ob.kickFunc()
}
