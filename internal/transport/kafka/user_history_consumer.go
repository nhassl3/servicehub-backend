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

const opUhc = "user-history-consumer: "

type UserHistoryConsumer struct {
	consumer *kafka.Consumer
	repo     domain.UserHistoryRepository
	log      *zap.Logger
}

func NewUserHistoryConsumer(
	consumer *kafka.Consumer,
	repo domain.UserHistoryRepository,
	log *zap.Logger,
) *UserHistoryConsumer {
	return &UserHistoryConsumer{
		consumer: consumer,
		repo:     repo,
		log:      log,
	}
}

func (c *UserHistoryConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx, c.handle)
}

func (c *UserHistoryConsumer) handle(ctx context.Context, msg segmentio.Message) error {
	var env domain.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		c.log.Error(opUhc+"bad envelope", zap.Error(err))
		return err
	}

	if err := c.mapEvent(ctx, env); err != nil {
		c.log.Error(opUhc+err.Error(), zap.Error(err))
		return err
	}

	return nil
}

func (c *UserHistoryConsumer) mapEvent(ctx context.Context, env domain.Envelope) error {
	switch env.Type {
	case domain.CreateUserHistory:
		var p domain.CreateUserHistoryParams
		if err := util.DecodePayload(env.Payload, &p); err != nil {
			c.log.Error(opUhc+"bad payload", zap.Error(err))
			return err
		}
		_, err := c.repo.CreateUserHistory(ctx, &p)
		if err != nil {
			c.log.Error(opUhc+"failed to create user history", zap.Error(err))
			return err
		}
	default:
		c.log.Debug(opUhc+"skipping unknown event type", zap.Int8("type", int8(env.Type)))
	}
	return nil
}
