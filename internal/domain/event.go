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
	_                    Type = iota << 1
	OrderCreated              // order.created
	OrderStatusChanged        // order.changed
	TransactionCreated        // transaction.created
	BalanceUpdated            // balance.updated
	IndexedProduct            // index.prouct.update_create
	DeletedProduct            // index.prouct.delete
	UserRegistered            // user.registered
	ProductStatusChanged      // product.status.changed
	ProductRatingChanged      // product.rating.changed
	ModerationApproved        // moderation.approved
	ModerationRejected        // moderation.rejected
	OrderItemCreated          // order.item.created
)

//go:generate mockgen -source=event.go -destination=../repository/mock/event_publisher_mock.go -package=mockrepo
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, payload OrderCreatedPayload) error
	PublishOrderStatusChanged(ctx context.Context, payload OrderStatusChangedPayload) error
	PublishTransactionCreated(ctx context.Context, payload TransactionCreatedPayload) error
	PublishBalanceUpdated(ctx context.Context, payload BalanceUpdatedPayload) error
	PublishIndexedProduct(ctx context.Context, product *Product) error
	PublishDeletedProduct(ctx context.Context, id string) error
	PublishUserRegistered(ctx context.Context, payload UserRegisteredPayload) error
	PublishProductStatusChanged(ctx context.Context, payload ProductStatusChangedPayload) error
	PublishProductRatingChanged(ctx context.Context, payload ProductRatingChangedPayload) error
	PublishModerationApproved(ctx context.Context, payload ModerationApprovedPayload) error
	PublishModerationRejected(ctx context.Context, payload ModerationRejectedPayload) error
	PublishOrderItemCreated(ctx context.Context, payload OrderItemCreatedPayload) error
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

// ── Analytics events ─────────────────────────────────────────────────────────

type UserRegisteredPayload struct {
	UID       string    `json:"uid"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type ProductStatusChangedPayload struct {
	ID           string    `json:"id"`
	SellerID     string    `json:"seller_id"`
	CategoryID   int       `json:"category_id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	Rating       float64   `json:"rating"`
	SalesCount   int       `json:"sales_count"`
	ReviewsCount int       `json:"reviews_count"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type ProductRatingChangedPayload struct {
	ID           string    `json:"id"`
	CategoryID   int       `json:"category_id"`
	Title        string    `json:"title"`
	Rating       float64   `json:"rating"`
	ReviewsCount int       `json:"reviews_count"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type ModerationApprovedPayload struct {
	ProductID     string    `json:"product_id"`
	CategoryID    int       `json:"category_id"`
	AdminID       string    `json:"admin_id"`
	AdminUsername string    `json:"admin_username"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type ModerationRejectedPayload struct {
	ProductID     string    `json:"product_id"`
	CategoryID    int       `json:"category_id"`
	AdminID       string    `json:"admin_id"`
	AdminUsername string    `json:"admin_username"`
	Reason        string    `json:"reason"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type OrderItemCreatedPayload struct {
	OrderID    string    `json:"order_id"`
	ProductID  string    `json:"product_id"`
	CategoryID int       `json:"category_id"`
	SellerID   string    `json:"seller_id"`
	Qty        int       `json:"qty"`
	Total      float64   `json:"total"`
	OccurredAt time.Time `json:"occurred_at"`
}
