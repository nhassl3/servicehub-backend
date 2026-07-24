package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nhassl3/servicehub-backend/pkg/kafka/util"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer публикует события в конкретный топик Kafka.
type Producer struct {
	writer *kafka.Writer
	log    *zap.Logger
}

// NewProducer создаёт продюсера для одного топика.
// Balancer=LeastBytes равномерно распределяет сообщения по партициям,
// RequireAll гарантирует подтверждение записи всеми синхронными репликами.
func NewProducer(brokers []string, topic string, log *zap.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
	}

	return &Producer{writer: w, log: log}
}

// Publish сериализует событие в JSON и отправляет его с заданным ключом
// (ключ определяет партицию — например, order_id или user_id для сохранения порядка).
func (p *Producer) Publish(ctx context.Context, key string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.Error("kafka: failed to publish message",
			zap.String("topic", p.writer.Topic),
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	p.log.Debug("kafka: message published",
		zap.String("topic", p.writer.Topic),
		zap.String("key", key),
	)
	return nil
}

// Close закрывает writer с ретраями: закрытие может временно упасть,
// если в моменте идёт фоновая отправка батча или сетевой сбой до брокера.
// kafka.Writer.Close() идемпотентен, поэтому повторный вызов безопасен.
func (p *Producer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attempt := 0
	if err := util.WithRetry(ctx, util.DefaultCloseRetry, func() error {
		attempt++
		if err := p.writer.Close(); err != nil {
			p.log.Warn("kafka: producer close attempt failed",
				zap.String("topic", p.writer.Topic),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			return err
		}
		return nil
	}); err != nil {
		p.log.Error("kafka: producer close failed after retries",
			zap.String("topic", p.writer.Topic),
			zap.Int("attempts", attempt),
			zap.Error(err),
		)
		return err
	}
	return nil
}
