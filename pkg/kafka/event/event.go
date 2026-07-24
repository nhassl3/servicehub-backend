package event

import "time"

type Type int8

// TODO: change type from string to bits
const (
	OrderCreated       Type = 1 // order.created
	OrderStatusChanged Type = 2 // order.changed
	TransactionCreated Type = 4 // transaction.created
	BalanceUpdated     Type = 8 // balance.updated
)

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
	OrderID  int64   `json:"order_id"`
	UserID   int64   `json:"user_id"`
	SellerID int64   `json:"seller_id"`
	Total    float64 `json:"total"`
}

type OrderStatusChangedPayload struct {
	OrderID   int64  `json:"order_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type TransactionCreatedPayload struct {
	TransactionID int64   `json:"transaction_id"`
	UserID        int64   `json:"user_id"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"` // deposit | withdrawal | payment
}

type BalanceUpdatedPayload struct {
	UserID     int64   `json:"user_id"`
	NewBalance float64 `json:"new_balance"`
}
