package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type ProducerPool struct {
	historyCh chan HistoryMessage
	dlqCh     chan DLQMessage
	writers   []*kafka.Writer
	dlqWriter *kafka.Writer
	wg        sync.WaitGroup
	logger    *slog.Logger
}

func NewProducerPool(brokers []string, logger *slog.Logger) *ProducerPool {
	pool := &ProducerPool{
		historyCh: make(chan HistoryMessage, 50000),
		dlqCh:     make(chan DLQMessage, 1000),
		logger:    logger,
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		w := newKafkaWriter(brokers, "chat-server-history")
		pool.writers = append(pool.writers, w)
	}

	pool.dlqWriter = newKafkaWriter(brokers, "chat-server-history-dlq")

	pool.Start()
	return pool
}

func newKafkaWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Async:        true,
		RequiredAcks: kafka.RequireOne,
		BatchSize:    500,
		BatchTimeout: 10 * time.Millisecond,
		Compression:  kafka.Snappy,
	}
}

func (p *ProducerPool) Post(msg HistoryMessage) error {
	select {
	case p.historyCh <- msg:
		return nil
	default:
		p.logger.Warn("main buffer full")
		dlqMsg := DLQMessage{
			HistoryMessage: msg,
			Error:          errors.New("main_buffer_is_full").Error(),
			Retry:          1,
		}
		return p.sendToDLQ(dlqMsg)
	}
}

func (p *ProducerPool) worker(w *kafka.Writer, id int) {
	defer p.wg.Done()
	defer w.Close()

	for msg := range p.historyCh {
		key := []byte(msg.ChatID + "_" + msg.UserID)
		if msgJson, err := msg.MarshalJSON(); err == nil {
			if err := w.WriteMessages(context.Background(),
				kafka.Message{Key: key, Value: msgJson},
			); err != nil {
				p.logger.Error("Kafka write failed", slog.Any("err", err))
				dlqMsg := DLQMessage{
					HistoryMessage: msg,
					Error:          err.Error(),
					Retry:          1,
				}
				p.sendToDLQ(dlqMsg)
			}
		} else {
			p.logger.Error("Message dropped", slog.Any("err", err))
		}

	}
}

func (p *ProducerPool) dlqWorker() {
	defer p.wg.Done()
	defer p.dlqWriter.Close()

	for dlqMsg := range p.dlqCh {
		key := []byte(fmt.Sprintf("%s_%s_dlq", dlqMsg.ChatID, dlqMsg.UserID))
		if msgJson, err := dlqMsg.MarshalJSON(); err == nil {
			if err := p.dlqWriter.WriteMessages(context.Background(),
				kafka.Message{
					Key:   key,
					Value: msgJson,
				},
			); err != nil {
				p.logger.Error("DLQ write failed", slog.Any("err", err))
				// Финальный drop или persistent queue
			}
		}

	}
}

func (p *ProducerPool) sendToDLQ(dlqMsg DLQMessage) error {
	select {
	case p.dlqCh <- dlqMsg:
		return nil
	default:
		p.logger.Error("DLQ buffer full — CRITICAL!")
		return fmt.Errorf("dlq_buffer_full")
	}
}

func (p *ProducerPool) Start() {
	p.wg.Add(len(p.writers) + 1) // + DLQ worker

	for i, w := range p.writers {
		go p.worker(w, i)
	}
	go p.dlqWorker()
}

func (p *ProducerPool) Close() {
	close(p.historyCh)
	close(p.dlqCh)
	p.wg.Wait()

	for _, w := range p.writers {
		w.Close()
	}
	p.dlqWriter.Close()
}
