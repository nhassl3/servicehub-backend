package kafka

import (
	"context"
	"encoding/json"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	"github.com/nhassl3/servicehub-backend/pkg/kafka/util"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// AnalyticsConsumer subscribes to all domain topics, normalizes the relevant
// events into analytics facts and batch-writes them to ClickHouse. Unfamiliar
// event types are ignored (mirrors NotificationConsumer). ClickHouse is not a
// source of truth — a failure here never blocks the write path, only logs.
type AnalyticsConsumer struct {
	consumer        *kafka.Consumer
	repo            domain.AnalyticsRepository
	categoriesRedis domain.CategoriesRedis
	log             *zap.Logger
}

func NewAnalyticsConsumer(
	consumer *kafka.Consumer,
	repo domain.AnalyticsRepository,
	categoriesRedis domain.CategoriesRedis,
	log *zap.Logger,
) *AnalyticsConsumer {
	return &AnalyticsConsumer{consumer: consumer, repo: repo, categoriesRedis: categoriesRedis, log: log}
}

func (c *AnalyticsConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx, c.handle)
}

func (c *AnalyticsConsumer) handle(ctx context.Context, msg segmentio.Message) error {
	var env domain.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		c.log.Error("analytics-consumer: bad envelope", zap.Error(err))
		return err
	}

	event, ok := c.mapEvent(env)
	if !ok {
		c.log.Debug("analytics-consumer: skipping unknown event type", zap.Int8("type", int8(env.Type)))
		return nil
	}

	if event.CategoryID > 0 {
		event.CategoryName = c.getCategoryName(ctx, event.CategoryID)
	}

	// ClickHouse writes are best-effort: retry briefly, then drop. Analytics
	// loss is acceptable — Postgres remains the source of truth.
	if err := util.WithRetry(ctx, util.DefaultCloseRetry, func() error {
		return c.repo.InsertEvents(ctx, []domain.AnalyticsEvent{event})
	}); err != nil {
		c.log.Debug("analytics-consumer: failed to insert events", zap.Error(err))
	}

	return nil
}

// getCategoryName returns category name by category id given from producer in decoded and mapped payload.
// WARN: always update categories redis storage cache when updates categories in main repository - database
// because it's get category by id and given new id not store in redis if not commited changes to redis storage.
func (c *AnalyticsConsumer) getCategoryName(ctx context.Context, categoryID int) string {
	categories, err := c.categoriesRedis.Categories(ctx)
	if err != nil {
		c.log.Warn("analytics-consumer: failed to get categories", zap.Error(err))
		return ""
	}
	category := categories.GetCategoryById(categoryID)
	if category == nil {
		c.log.Warn("analytics-consumer: failed to get category by ID", zap.Int("ID", categoryID))
		return ""
	}
	return category.Name
}

// mapEvent converts an envelope into a normalized AnalyticsEvent. ok=false for
// event types the consumer does not track. A decoded blank EventType signals a
// malformed payload for a tracked type.
func (c *AnalyticsConsumer) mapEvent(env domain.Envelope) (domain.AnalyticsEvent, bool) {
	switch env.Type {
	case domain.UserRegistered:
		var p domain.UserRegisteredPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt: p.CreatedAt,
			EventType:  domain.UserRegisteredEventType,
		}, true

	case domain.ProductStatusChanged:
		var p domain.ProductStatusChangedPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt:   p.OccurredAt,
			EventType:    domain.ProductStatusChangedEventType,
			ProductID:    p.ID,
			SellerID:     p.SellerID,
			CategoryID:   p.CategoryID,
			Title:        p.Title,
			Status:       p.Status,
			Rating:       p.Rating,
			SalesCount:   p.SalesCount,
			ReviewsCount: p.ReviewsCount,
		}, true

	case domain.ProductRatingChanged:
		var p domain.ProductRatingChangedPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt:   p.OccurredAt,
			EventType:    domain.ProductRatingChangedEventType,
			ProductID:    p.ID,
			CategoryID:   p.CategoryID,
			Title:        p.Title,
			Rating:       p.Rating,
			ReviewsCount: p.ReviewsCount,
		}, true

	case domain.ModerationApproved:
		var p domain.ModerationApprovedPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt:    p.OccurredAt,
			EventType:     domain.ModerationApprovedEventType,
			ProductID:     p.ProductID,
			CategoryID:    p.CategoryID,
			AdminID:       p.AdminID,
			AdminUsername: p.AdminUsername,
		}, true

	case domain.ModerationRejected:
		var p domain.ModerationRejectedPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt:    p.OccurredAt,
			EventType:     domain.ModerationRejectedEventType,
			ProductID:     p.ProductID,
			CategoryID:    p.CategoryID,
			AdminID:       p.AdminID,
			AdminUsername: p.AdminUsername,
			Reason:        p.Reason,
		}, true

	case domain.OrderItemCreated:
		var p domain.OrderItemCreatedPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt: p.OccurredAt,
			EventType:  domain.OrderItemCreatedEventType,
			ProductID:  p.ProductID,
			CategoryID: p.CategoryID,
			Title:      p.Title,
			SellerID:   p.SellerID,
			OrderID:    p.OrderID,
			Quantity:   p.Qty,
			Total:      p.Total,
		}, true
	default:
		return domain.AnalyticsEvent{}, false
	}
}
