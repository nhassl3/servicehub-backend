package mock

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// NoopPublisher is a trivial EventPublisher stub so existing auth tests don't
// need a gomock controller.
type NoopPublisher struct{}

func (NoopPublisher) PublishOrderCreated(context.Context, domain.OrderCreatedPayload) error {
	return nil
}
func (NoopPublisher) PublishOrderStatusChanged(context.Context, domain.OrderStatusChangedPayload) error {
	return nil
}
func (NoopPublisher) PublishTransactionCreated(context.Context, domain.TransactionCreatedPayload) error {
	return nil
}
func (NoopPublisher) PublishBalanceUpdated(context.Context, domain.BalanceUpdatedPayload) error {
	return nil
}
func (NoopPublisher) PublishIndexedProduct(context.Context, *domain.Product) error { return nil }
func (NoopPublisher) PublishDeletedProduct(context.Context, string) error          { return nil }
func (NoopPublisher) PublishUserRegistered(context.Context, domain.UserRegisteredPayload) error {
	return nil
}
func (NoopPublisher) PublishProductStatusChanged(context.Context, domain.ProductStatusChangedPayload) error {
	return nil
}
func (NoopPublisher) PublishCreateUserHistory(context.Context, domain.CreateUserHistoryParams) error {
	return nil
}
func (NoopPublisher) PublishProductRatingChanged(context.Context, domain.ProductRatingChangedPayload) error {
	return nil
}
func (NoopPublisher) PublishModerationApproved(context.Context, domain.ModerationApprovedPayload) error {
	return nil
}
func (NoopPublisher) PublishModerationRejected(context.Context, domain.ModerationRejectedPayload) error {
	return nil
}
func (NoopPublisher) PublishOrderItemCreated(context.Context, domain.OrderItemCreatedPayload) error {
	return nil
}
func (NoopPublisher) Close() error { return nil }
