package domain

import (
	"context"
	"time"
)

// EventType types for analytics
type EventType string

const (
	UserRegisteredEventType       EventType = "user_registered"
	ProductStatusChangedEventType EventType = "product_status_changed"
	ProductRatingChangedEventType EventType = "product_rating_changed"
	ModerationApprovedEventType   EventType = "moderation.approved"
	ModerationRejectedEventType   EventType = "moderation.rejected"
	OrderItemCreatedEventType     EventType = "order_item_created"
)

// AnalyticsEvent is a single normalized row written to the ClickHouse
// fact table (analytics.events). Postgres remains the source of truth;
// these rows are denormalized OLAP facts consumed from Kafka events.
type AnalyticsEvent struct {
	OccurredAt    time.Time
	EventType     EventType // 'user_registered' | 'product_status_changed' | ...
	ProductID     string
	CategoryID    int
	Title         string
	SellerID      string
	AdminID       string
	AdminUsername string
	Status        string // draft | active | inactive
	Rating        float64
	ReviewsCount  int
	SalesCount    int
	Reason        string
	OrderID       string
	Quantity      int
	Total         float64
}

// AdminStatisticsParams narrows the requested analytics period/granularity.
type AdminStatisticsParams struct {
	From        time.Time
	To          time.Time
	Granularity string // "day" | "hour", default "day"
}

// AdminStatistics is the aggregate dashboard response for an admin.
type AdminStatistics struct {
	Products      ProductStatusStats
	TopProducts   []TopProduct
	TopCategories []CategorySales
	Registrations []RegistrationPoint
	Moderations   []ModerationPoint
}

// ProductStatusStats counts products grouped by their current moderation
// outcome: verified = active, pending = draft, rejected = inactive.
type ProductStatusStats struct {
	Verified int
	Pending  int
	Rejected int
}

// TopProduct is the top-rated product within the period.
type TopProduct struct {
	ID           string
	Title        string
	CategoryID   int
	Rating       float64
	SalesCount   int
	ReviewsCount int
}

// CategorySales aggregates sales (quantity) by category within the period.
type CategorySales struct {
	CategoryID int
	Name       string
	SalesCount int
	Total      float64
}

// RegistrationPoint buckets user registrations by day/hour.
type RegistrationPoint struct {
	Bucket time.Time
	Count  int
}

// ModerationPoint buckets moderation actions by day/hour, split by admin.
type ModerationPoint struct {
	Bucket        time.Time
	Count         int
	AdminID       string
	AdminUsername string
}

//go:generate mockgen -source=analytics.go -destination=../repository/mock/analytics_repo_mock.go -package=mockrepo
type AnalyticsRepository interface {
	InsertEvents(ctx context.Context, events []AnalyticsEvent) error
	GetAdminStatistics(ctx context.Context, params AdminStatisticsParams) (*AdminStatistics, error)
}
