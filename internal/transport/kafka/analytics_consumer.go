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
	consumer *kafka.Consumer
	repo     domain.AnalyticsRepository
	log      *zap.Logger
}

func NewAnalyticsConsumer(consumer *kafka.Consumer, repo domain.AnalyticsRepository, log *zap.Logger) *AnalyticsConsumer {
	return &AnalyticsConsumer{consumer: consumer, repo: repo, log: log}
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
	if event.EventType == "" {
		// known type but bad payload — keep the consumer alive, drop the message.
		return nil
	}

	// ClickHouse writes are best-effort: retry briefly, then drop. Analytics
	// loss is acceptable — Postgres remains the source of truth.
	_ = util.WithRetry(ctx, util.DefaultCloseRetry, func() error {
		return c.repo.InsertEvents(ctx, []domain.AnalyticsEvent{event})
	})
	return nil
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
			EventType:  "user_registered",
		}, true

	case domain.ProductStatusChanged:
		var p domain.ProductStatusChangedPayload
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			return domain.AnalyticsEvent{}, false
		}
		return domain.AnalyticsEvent{
			OccurredAt:   p.OccurredAt,
			EventType:    "product_status_changed",
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
			EventType:    "product_rating_changed",
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
			EventType:     "moderation.approved",
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
			EventType:     "moderation.rejected",
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
			EventType:  "order_item_created",
			ProductID:  p.ProductID,
			CategoryID: p.CategoryID,
			SellerID:   p.SellerID,
			OrderID:    p.OrderID,
			Quantity:   p.Qty,
			Total:      p.Total,
		}, true
	}

	return domain.AnalyticsEvent{}, false
}
