package domain

import (
	"context"
	"time"
)

type Type int8

// TODO: change type from string to bits
const (
	OrderCreated       Type = 1 // order.created
	OrderStatusChanged Type = 2 // order.changed
	TransactionCreated Type = 4 // transaction.created
	BalanceUpdated     Type = 8 // balance.updated
)

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, payload OrderCreatedPayload) error
	PublishOrderStatusChanged(ctx context.Context, payload OrderStatusChangedPayload) error
	PublishTransactionCreated(ctx context.Context, payload TransactionCreatedPayload) error
	PublishBalanceUpdated(ctx context.Context, payload BalanceUpdatedPayload) error
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
	Total    float64 `json:"total"`
}

type OrderStatusChangedPayload struct {
	OrderUID  string `json:"order_uid"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type TransactionCreatedPayload struct {
	TransactionID int64   `json:"transaction_id"`
	Username      string  `json:"username"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"` // deposit | withdrawal | payment
}

type BalanceUpdatedPayload struct {
	Username   string  `json:"username"`
	NewBalance float64 `json:"new_balance"`
}
