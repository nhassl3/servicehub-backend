package kafka

import (
	"context"
	"encoding/json"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/nhassl3/servicehub-backend/pkg/kafka/util"
)

// NotificationConsumer подписывается на доменные события и превращает
// их в уведомления пользователю (email/push/websocket — реализация подставляется через notifier).
type NotificationConsumer struct {
	consumer *Consumer
	notifier Notifier
	log      *zap.Logger
}

// Notifier — интерфейс, который реализуется конкретным каналом доставки.
// Сюда позже подключается, например, email-сервис или push-сервис.
type Notifier interface {
	Notify(ctx context.Context, userID int64, title, body string) error
}

func NewNotificationConsumer(consumer *Consumer, notifier Notifier, log *zap.Logger) *NotificationConsumer {
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
	return c.notifier.Notify(ctx, payload.UserID, "Заказ оформлен",
		"Ваш заказ успешно создан и передан в обработку.")
}

func (c *NotificationConsumer) handleOrderStatusChanged(ctx context.Context, env domain.Envelope) error {
	var payload domain.OrderStatusChangedPayload
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		return err
	}
	return c.notifier.Notify(ctx, 0, "Статус заказа изменён",
		"Заказ #"+util.Itoa(payload.OrderID)+" теперь: "+payload.NewStatus)
}

func (c *NotificationConsumer) handleTransactionCreated(ctx context.Context, env domain.Envelope) error {
	var payload domain.TransactionCreatedPayload
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		return err
	}
	return c.notifier.Notify(ctx, payload.UserID, "Новая транзакция",
		"Операция на сумму "+util.Ftoa(payload.Amount)+" зафиксирована.")
}
