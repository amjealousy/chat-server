package utils

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/sqids/sqids-go"
)

type IDGen struct {
	sqids *sqids.Sqids
	mu    sync.Mutex
	seq   uint64
}

var IDGenerator *IDGen

func NewIDGen() *IDGen {
	symbols := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	opts := sqids.Options{
		Alphabet:  symbols,
		MinLength: 8,
	}
	s, _ := sqids.New(opts)

	return &IDGen{sqids: s}
}

func (g *IDGen) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := uint64(time.Now().UnixNano() / 1e6)
	idNum := now<<32 | g.seq
	g.seq++

	ids, _ := g.sqids.Encode([]uint64{idNum})
	return strconv.Itoa(int(ids[0]))
}

func LogInfoWithReqID(
	ctx context.Context,
	logger *slog.Logger,
	reqID string,
	msg string,
	attrs ...any, // additional parameters
) {
	base := []any{
		"request_id", reqID,
	}
	base = append(base, attrs...)

	logger.InfoContext(ctx, msg, base...)
}
