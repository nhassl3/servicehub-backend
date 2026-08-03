package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mockrepo "github.com/nhassl3/servicehub-backend/internal/repository/mock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/golang/mock/gomock"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

func newAnalyticsConsumer(t *testing.T) (*AnalyticsConsumer, *mockrepo.MockAnalyticsRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mockrepo.NewMockAnalyticsRepository(ctrl)
	return NewAnalyticsConsumer(nil, repo, zap.NewNop()), repo
}

func envelopeMsg(t *testing.T, env domain.Envelope) *kafka.Message {
	t.Helper()
	b, err := json.Marshal(env)
	require.NoError(t, err)
	return &kafka.Message{Value: b}
}

func TestAnalyticsConsumer_mapEvent_UnknownType(t *testing.T) {
	c, _ := newAnalyticsConsumer(t)
	_, ok := c.mapEvent(domain.Envelope{Type: domain.OrderCreated})
	require.False(t, ok)
}

func TestAnalyticsConsumer_mapEvent_OrderItemCreated(t *testing.T) {
	c, _ := newAnalyticsConsumer(t)
	env := domain.NewEnvelope(domain.OrderItemCreated, domain.OrderItemCreatedPayload{
		OrderID:    "ord-1",
		ProductID:  "prod-1",
		CategoryID: 7,
		SellerID:   "seller-1",
		Qty:        2,
		Total:      149.99,
		OccurredAt: time.Now(),
	})
	event, ok := c.mapEvent(env)
	require.True(t, ok)
	require.Equal(t, "order_item_created", event.EventType)
	require.Equal(t, "prod-1", event.ProductID)
	require.Equal(t, 7, event.CategoryID)
	require.Equal(t, 2, event.Quantity)
}

func TestAnalyticsConsumerHandle_SkipsUnknownType(t *testing.T) {
	c, repo := newAnalyticsConsumer(t)
	ctx := context.Background()
	repo.EXPECT().InsertEvents(ctx, gomock.Any()).Times(0)

	env := domain.NewEnvelope(domain.IndexedProduct, "anything")
	err := c.handle(ctx, *envelopeMsg(t, env))
	require.NoError(t, err)
}

func TestAnalyticsConsumerHandle_InsertsOrderItem(t *testing.T) {
	c, repo := newAnalyticsConsumer(t)
	ctx := context.Background()
	repo.EXPECT().InsertEvents(ctx, gomock.Any()).Return(nil)

	env := domain.NewEnvelope(domain.OrderItemCreated, domain.OrderItemCreatedPayload{
		OrderID:    "ord-1",
		ProductID:  "prod-1",
		CategoryID: 3,
		Qty:        1,
		Total:      99,
		OccurredAt: time.Now(),
	})
	err := c.handle(ctx, *envelopeMsg(t, env))
	require.NoError(t, err)
}
