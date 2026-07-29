package kafka

import (
	"context"

	segmentio "github.com/segmentio/kafka-go"
)

type ConsumerHandler interface {
	Run(ctx context.Context) error
	handle(ctx context.Context, msg segmentio.Message) error
}
