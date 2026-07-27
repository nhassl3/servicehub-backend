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
	dlq    *Producer
	log    *zap.Logger
}

func NewConsumer(brokers []string, topic, groupID string, log *zap.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID, // consumer group — офсеты хранятся в Kafka, ребалансировка партий между инстансами
		MinBytes: 1,       // не ждать накопления батча — топики с низким трафиком (order/tx events) должны доставляться сразу
		MaxBytes: 10e6,    // 10MB — потолок на случай крупных сообщений
		MaxWait:  1 * time.Second,
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
			c.log.Error("kafka: fetch message failed", zap.String("topic", c.reader.Config().Topic), zap.Error(err))
			continue
		}

		c.log.Debug("kafka: message fetched",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)

		handlerErr := util.WithRetry(ctx, util.RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   200 * time.Millisecond,
			MaxDelay:    2 * time.Second,
		}, func() error {
			return handler(ctx, msg)
		})
		if handlerErr != nil {
			c.log.Error("kafka: handler failed after retries",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(handlerErr),
			)

			if c.dlq != nil {
				if dlqErr := c.sendToDLQ(ctx, msg, handlerErr); dlqErr != nil {
					c.log.Error("kafka: failed to publish to DLQ, will retry this message on next poll",
						zap.String("topic", msg.Topic), zap.Error(dlqErr))
					continue // offset will not commit - try everyone on beginning next iteration
				}
				c.log.Warn("kafka: message moved to DLQ", zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset))
			} else {
				// without dlq - delete partition but very noise logging this failure
				c.log.Warn("kafka: no DLQ configured, message dropped after exhausting retries",
					zap.String("topic", msg.Topic), zap.Int64("offset", msg.Offset))
			}
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.log.Error("kafka: commit failed", zap.Error(err))
		} else {
			c.log.Debug("kafka: offset committed", zap.String("topic", msg.Topic), zap.Int64("offset", msg.Offset))
		}
	}
}

// dlqMessage — конверт для DLQ-топика: исходное сообщение плюс контекст сбоя,
// достаточный для расследования без похода в логи.
type dlqMessage struct {
	OriginalTopic string    `json:"original_topic"`
	Partition     int       `json:"partition"`
	Offset        int64     `json:"offset"`
	Key           string    `json:"key"`
	Value         string    `json:"value"` // исходный payload как есть (уже JSON-строка)
	Error         string    `json:"error"`
	FailedAt      time.Time `json:"failed_at"`
}

func (c *Consumer) sendToDLQ(ctx context.Context, msg kafka.Message, handlerErr error) error {
	dlq := dlqMessage{
		OriginalTopic: msg.Topic,
		Partition:     msg.Partition,
		Offset:        msg.Offset,
		Key:           string(msg.Key),
		Value:         string(msg.Value),
		Error:         handlerErr.Error(),
		FailedAt:      time.Now(),
	}
	return c.dlq.Publish(ctx, string(msg.Key), dlq)
}

// WithDLQ прикрепляет продюсера DLQ-топика к консьюмеру. Без вызова этого метода
// сообщения, не обработанные после всех ретраев, будут закоммичены и потеряны —
// с явным warn-логом об этом, но без остановки партиции.
func (c *Consumer) WithDLQ(dlqProducer *Producer) *Consumer {
	c.dlq = dlqProducer
	return c
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
