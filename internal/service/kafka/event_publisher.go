package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
)

// EventPublisher инкапсулирует продюсеров по топикам и даёт доменным
// сервисам простой API для публикации событий, не завязанный на детали Kafka.
type EventPublisher struct {
	eventKafkaProducers map[string]*kafka.Producer
}

func NewEventPublisher(eventKafkaProducers map[string]*kafka.Producer) *EventPublisher {
	return &EventPublisher{
		eventKafkaProducers: eventKafkaProducers,
	}
}

func (p *EventPublisher) PublishOrderCreated(ctx context.Context, payload domain.OrderCreatedPayload) error {
	env := domain.NewEnvelope(domain.OrderCreated, payload)
	return p.publish(ctx, domain.TopicOrderEvent, fmt.Sprintf("order-%s", payload.OrderUID), env)
}

func (p *EventPublisher) PublishOrderStatusChanged(ctx context.Context, payload domain.OrderStatusChangedPayload) error {
	env := domain.NewEnvelope(domain.OrderStatusChanged, payload)
	return p.publish(ctx, domain.TopicOrderEvent, fmt.Sprintf("order-%s", payload.OrderUID), env)
}

func (p *EventPublisher) PublishTransactionCreated(ctx context.Context, payload domain.TransactionCreatedPayload) error {
	env := domain.NewEnvelope(domain.TransactionCreated, payload)
	return p.publish(ctx, domain.TopicTransactionEvent, fmt.Sprintf("user-%s", payload.Username), env)
}

func (p *EventPublisher) PublishBalanceUpdated(ctx context.Context, payload domain.BalanceUpdatedPayload) error {
	env := domain.NewEnvelope(domain.BalanceUpdated, payload)
	return p.publish(ctx, domain.TopicTransactionEvent, fmt.Sprintf("user-%s", payload.Username), env)
}

func (p *EventPublisher) PublishIndexedProduct(ctx context.Context, product *domain.Product) error {
	env := domain.NewEnvelope(domain.IndexedProduct, product)
	return p.publish(ctx, domain.TopicProductEvent, fmt.Sprintf("indexed-product-%s", product.ID), env)
}

func (p *EventPublisher) PublishDeletedProduct(ctx context.Context, id string) error {
	env := domain.NewEnvelope(domain.DeletedProduct, id)
	return p.publish(ctx, domain.TopicProductEvent, fmt.Sprintf("deleted-product-%s", id), env)
}

// publish public event in producer
func (p *EventPublisher) publish(ctx context.Context, topic domain.Topic, key string, event interface{}) error {
	return p.eventKafkaProducers[string(topic)].Publish(ctx, key, event)
}

// Close закрывает оба продюсера независимо: ошибка закрытия одного
// не должна помешать попытке закрыть второй (утечка соединения хуже, чем один лишний вызов).
func (p *EventPublisher) Close() error {
	var producerErrors error
	for _, producer := range p.eventKafkaProducers {
		producerErrors = errors.Join(producer.Close())
	}
	return producerErrors
}
