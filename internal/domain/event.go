package domain

import (
	"context"
	"time"
)

type Topic string

const (
	TopicOrderEvent       Topic = "order_events"
	TopicProductEvent     Topic = "product_events"
	TopicTransactionEvent Topic = "transaction_events"
)

var (
	DLQTopics = []Topic{
		TopicOrderEvent,
	}
)

type Type int8

const (
	_                  Type = iota << 1
	OrderCreated            // order.created
	OrderStatusChanged      // order.changed
	TransactionCreated      // transaction.created
	BalanceUpdated          // balance.updated
	IndexedProduct          // index.prouct.update_create
	DeletedProduct          // index.prouct.delete
)

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, payload OrderCreatedPayload) error
	PublishOrderStatusChanged(ctx context.Context, payload OrderStatusChangedPayload) error
	PublishTransactionCreated(ctx context.Context, payload TransactionCreatedPayload) error
	PublishBalanceUpdated(ctx context.Context, payload BalanceUpdatedPayload) error
	PublishIndexedProduct(ctx context.Context, product *Product) error
	PublishDeletedProduct(ctx context.Context, id string) error
	Close() error
}

// Envelope — единый конверт для всех событий, чтобы консьюмеры
// могли определить тип события до разбора payload.
type Envelope struct {
	Type       Type        `json:"type"`
	OccurredAt time.Time   `json:"occurred_at"`
	Payload    interface{} `json:"payload"`
}

func NewEnvelope(t Type, payload interface{}) Envelope {
	return Envelope{
		Type:       t,
		OccurredAt: time.Now(),
		Payload:    payload,
	}
}

type OrderCreatedPayload struct {
	OrderUID string  `json:"order_uid"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Total    float64 `json:"total"`
}

type OrderStatusChangedPayload struct {
	Email     string `json:"email"`
	OrderUID  string `json:"order_uid"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type TransactionCreatedPayload struct {
	Email         string  `json:"email"`
	TransactionID int64   `json:"transaction_id"`
	Username      string  `json:"username"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"` // deposit | withdrawal | payment
}

type BalanceUpdatedPayload struct {
	Email      string  `json:"email"`
	Username   string  `json:"username"`
	Amount     float64 `json:"amount"`
	NewBalance float64 `json:"new_balance"`
}
