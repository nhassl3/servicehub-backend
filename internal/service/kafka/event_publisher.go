package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	"github.com/nhassl3/servicehub-backend/pkg/kafka/event"
)

// EventPublisher инкапсулирует продюсеров по топикам и даёт доменным
// сервисам простой API для публикации событий, не завязанный на детали Kafka.
type EventPublisher struct {
	orderProducer       *kafka.Producer
	transactionProducer *kafka.Producer
}

func NewEventPublisher(orderProducer, transactionProducer *kafka.Producer) *EventPublisher {
	return &EventPublisher{
		orderProducer:       orderProducer,
		transactionProducer: transactionProducer,
	}
}

func (p *EventPublisher) PublishOrderCreated(ctx context.Context, payload event.OrderCreatedPayload) error {
	env := event.NewEnvelope(event.OrderCreated, payload)
	return p.orderProducer.Publish(ctx, fmt.Sprintf("order-%d", payload.OrderID), env)
}

func (p *EventPublisher) PublishOrderStatusChanged(ctx context.Context, payload event.OrderStatusChangedPayload) error {
	env := event.NewEnvelope(event.OrderStatusChanged, payload)
	return p.orderProducer.Publish(ctx, fmt.Sprintf("order-%d", payload.OrderID), env)
}

func (p *EventPublisher) PublishTransactionCreated(ctx context.Context, payload event.TransactionCreatedPayload) error {
	env := event.NewEnvelope(event.TransactionCreated, payload)
	return p.transactionProducer.Publish(ctx, fmt.Sprintf("user-%d", payload.UserID), env)
}

func (p *EventPublisher) PublishBalanceUpdated(ctx context.Context, payload event.BalanceUpdatedPayload) error {
	env := event.NewEnvelope(event.BalanceUpdated, payload)
	return p.transactionProducer.Publish(ctx, fmt.Sprintf("user-%d", payload.UserID), env)
}

// Close закрывает оба продюсера независимо: ошибка закрытия одного
// не должна помешать попытке закрыть второй (утечка соединения хуже, чем один лишний вызов).
func (p *EventPublisher) Close() error {
	orderErr := p.orderProducer.Close()
	txErr := p.transactionProducer.Close()
	return errors.Join(orderErr, txErr)
}
