package test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wsl.test/chat"
	"wsl.test/internal"
	"wsl.test/network"
	"wsl.test/processor"
	"wsl.test/utils"
)

var Controller *chat.Controller
var logger *slog.Logger

func CreateTestMember(userId string, chatID string) *chat.Member {
	fqdn := fmt.Sprintf("%s@test.domain", userId)
	user := chat.User{ID: userId, FQDN: fqdn}
	recv := make(chan internal.Message, 10)
	testConnection := &network.APIConnection{ID: userId, User: &user, ReceiveCh: recv}
	member := chat.NewMember(chatID, userId)
	member.AddConnection(testConnection)
	return member
}

func TestMain(m *testing.M) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger = slog.New(handler)
	_, file, _, _ := runtime.Caller(1)
	logger.WithGroup("test").
		With("file", filepath.Base(file))
	utils.IDGenerator = utils.NewIDGen()
	pool := processor.NewProducerPool([]string{"test"}, false)
	Controller = chat.NewController(os.Stdout, pool)
	go Controller.Start(context.Background())

	slog.Info("Reach the end of TestMain")
	os.Exit(m.Run())
}
func TestChat(t *testing.T) {
	Controller.CreateChat("1")
	chat := Controller.Chats["1"]
	defer chat.StopChat()
	const count = 10
	for i := range count {
		chat.Members[strconv.Itoa(i)] = CreateTestMember(strconv.Itoa(i), chat.ID)
	}
	logger.Info("List of members: ", slog.Any("members", chat.Members))
	var messages = [count]internal.TypeMessage{}
	for i := range count {
		message := internal.TypeMessage{Type: internal.Broadcast, Message: internal.Message{ChatID: chat.ID, From: strconv.Itoa(i), Cseq: strconv.Itoa(i), Text: "Test"}}
		messages[i] = message
	}
	messageCounterSend := atomic.Uint64{}
	messageCounterSend.Store(0)
	group := sync.WaitGroup{}
	group.Add(2)
	go func() {
		defer group.Done()
		for _, msg := range messages {
			messageCounterSend.Add(1)
			logger.Info("send a message: ", slog.Any("BroadcastMsg", msg))
			chat.MessageChan <- msg
		}

	}()
	messageCounterReceive := atomic.Uint64{}
	messageCounterReceive.Store(0)
	ticker := time.Tick(time.Second * 5)
	logger.Info("Start receiving messages: ", slog.Any("time before ", time.Now()))
	go func() {
		defer group.Done()
		const numMembers = 10
		innerGroup := sync.WaitGroup{}
		innerGroup.Add(numMembers)
		for id, mem := range chat.Members {
			go func() {
				defer innerGroup.Done()
				for range 9 {
					select {
					case msg := <-mem.Connection.ReceiveChan():
						fmt.Println("[ReceivingLoop] Received message: ", msg)
						messageCounterReceive.Add(1)
					case <-ticker:
						logger.Info("out of Ticker", slog.Any("time after", time.Now()), slog.Any("memberID", id))
						return

					}
				}

			}()

		}
		innerGroup.Wait()
	}()
	group.Wait()

	t.Log("Test result:", messageCounterReceive.Load())
	separator := strings.Repeat("=", 20)
	t.Log(separator)
	t.Logf("Received: %d (expected 10)", len(chat.Members))
	require.Equal(t, 10, len(chat.Members), "Members must be 10")
	if assert.Equal(t, uint64(90), messageCounterReceive.Load()) {
		t.Log("✅ messageCounterReceive OK")
	} else {
		t.Logf("[err] Expected 90, got %d", messageCounterReceive.Load())
	}

}
