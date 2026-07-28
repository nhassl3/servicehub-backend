package kafka

import (
	"context"
	"encoding/json"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/repository/elasticsearch"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	"github.com/nhassl3/servicehub-backend/pkg/kafka/util"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// ProductConsumer subscribe on domain events with Product and send to Elasticsearch instruction what to do with them
type ProductConsumer struct {
	consumer    *kafka.Consumer
	elasticRepo *elasticsearch.ProductESRepo
	log         *zap.Logger
}

func NewProductConsumer(consumer *kafka.Consumer, elasticRepo *elasticsearch.ProductESRepo, log *zap.Logger) *ProductConsumer {
	return &ProductConsumer{consumer: consumer, elasticRepo: elasticRepo, log: log}
}

func (c *ProductConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx, c.handle)
}

func (c *ProductConsumer) handle(ctx context.Context, msg segmentio.Message) error {
	var env domain.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		c.log.Error("product-consumer: bad envelope", zap.Error(err))
		return err
	}

	switch env.Type {
	case domain.IndexedProduct:
		return c.handleIndexedProduct(ctx, env)
	case domain.DeletedProduct:
		return c.handleDeletedProduct(ctx, env)
	default:
		c.log.Debug("product-consumer: skipping unknown event type", zap.Int8("type", int8(env.Type)))
		return nil
	}
}

func (c *ProductConsumer) handleIndexedProduct(ctx context.Context, env domain.Envelope) error {
	var payload domain.Product
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		c.log.Error("product-consumer: bad payload", zap.Error(err))
		return err
	}
	return c.elasticRepo.IndexProduct(ctx, &payload)
}

func (c *ProductConsumer) handleDeletedProduct(ctx context.Context, env domain.Envelope) error {
	var payload string
	if err := util.DecodePayload(env.Payload, &payload); err != nil {
		c.log.Error("product-consumer: bad payload", zap.Error(err))
		return err
	}
	return c.elasticRepo.DeleteProductIndex(ctx, payload)
}
