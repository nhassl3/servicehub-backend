package kafka

import (
	"context"
	"errors"
	"time"

	"github.com/nhassl3/servicehub-backend/pkg/kafka/util"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Handler обрабатывает одно сообщение. Возврат ошибки означает,
// что офсет коммитить не нужно (сообщение будет прочитано снова).
type Handler func(ctx context.Context, msg kafka.Message) error

// Consumer читает сообщения из одного топика в составе consumer group.
type Consumer struct {
	reader *kafka.Reader
	log    *zap.Logger
}

func NewConsumer(brokers []string, topic, groupID string, log *zap.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID, // consumer group — офсеты хранятся в Kafka, ребалансировка партий между инстансами
		MinBytes: 10e3,    // 10KB
		MaxBytes: 10e6,    // 10MB
	})

	return &Consumer{reader: r, log: log}
}

// Run блокирует выполнение и обрабатывает сообщения до отмены ctx.
// Коммит офсета выполняется вручную после успешной обработки (at-least-once).
func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("kafka: fetch message failed", zap.Error(err))
			continue
		}

		if err := handler(ctx, msg); err != nil {
			c.log.Error("kafka: handler failed",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
			// TODO: здесь можно добавить retry с backoff или отправку в DLQ-топик
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.log.Error("kafka: commit failed", zap.Error(err))
		}
	}
}

// Close закрывает reader с ретраями. Reader.Close() отменяет активный FetchMessage
// и должен быть идемпотентным, но при гонке с ребалансировкой consumer group
// первая попытка иногда возвращает ошибку — повторяем перед тем как сдаться.
func (c *Consumer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attempt := 0
	if err := util.WithRetry(ctx, util.DefaultCloseRetry, func() error {
		attempt++
		if err := c.reader.Close(); err != nil {
			c.log.Warn("kafka: consumer close attempt failed",
				zap.String("topic", c.reader.Config().Topic),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			return err
		}
		return nil
	}); err != nil {
		c.log.Error("kafka: consumer close failed after retries",
			zap.String("topic", c.reader.Config().Topic),
			zap.Int("attempts", attempt),
			zap.Error(err),
		)
		return err
	}
	return nil
}
