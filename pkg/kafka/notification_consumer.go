package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/nhassl3/servicehub-backend/pkg/kafka/util"
)

// NotificationConsumer подписывается на доменные события и превращает
// их в уведомления пользователю (email/push/websocket — реализация подставляется через notifier).
type NotificationConsumer struct {
	consumer *Consumer
	notifier mailer.Notifier
	log      *zap.Logger
}

func NewNotificationConsumer(consumer *Consumer, notifier mailer.Notifier, log *zap.Logger) *NotificationConsumer {
	return &NotificationConsumer{consumer: consumer, notifier: notifier, log: log}
}

func (c *NotificationConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx, c.handle)
}

func (c *NotificationConsumer) handle(ctx context.Context, msg segmentio.Message) error {
	var env domain.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		c.log.Error("notification-consumer: bad envelope", zap.Error(err))
		return err
	}

	switch env.Type {
	case domain.OrderCreated:
		return c.handleOrderCreated(ctx, env)
	case domain.OrderStatusChanged:
		return c.handleOrderStatusChanged(ctx, env)
	case domain.TransactionCreated:
		return c.handleTransactionCreated(ctx, env)
	default:
		c.log.Debug("notification-consumer: skipping unknown event type", zap.Int8("type", int8(env.Type)))
		return nil
	}
}

func (c *NotificationConsumer) handleOrderCreated(ctx context.Context, env domain.Envelope) error {
	var payload domain.OrderCreatedPayload
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		return err
	}
	return c.notifier.NotifyAnyMessage(ctx, "Заказ оформлен",
		fmt.Sprintf(
			"%s, Ваш заказ #%s на сумму $ %.2f успешно создан и передан в обработку.",
			payload.Username, payload.OrderUID, payload.Total), payload.Email)
}

func (c *NotificationConsumer) handleOrderStatusChanged(ctx context.Context, env domain.Envelope) error {
	var payload domain.OrderStatusChangedPayload
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		return err
	}
	return c.notifier.NotifyAnyMessage(ctx, "Статус заказа изменён",
		fmt.Sprintf("Статус заказа #%s изменился:\n%s ⟶ %s", payload.OrderUID, payload.OldStatus, payload.NewStatus), payload.Email)
}

func (c *NotificationConsumer) handleTransactionCreated(ctx context.Context, env domain.Envelope) error {
	var payload domain.TransactionCreatedPayload
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		return err
	}
	return c.notifier.NotifyAnyMessage(ctx, "Новая транзакция",
		fmt.Sprintf("Зафиксирована операция на сумму $ %.2f", payload.Amount), payload.Email)
}
