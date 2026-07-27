package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// WaitReady блокируется, пока брокер не подтвердит готовность через реального
// контроллера, либо пока не истечёт timeout. depends_on: service_healthy в
// docker-compose гарантирует только то, что брокер отвечает на базовые запросы —
// этого недостаточно сразу после холодного старта (Kafka в фоне ещё может
// создавать __consumer_offsets и выбирать для него лидеров), поэтому нужна
// отдельная проверка на уровне приложения перед тем, как создавать consumer group.
func WaitReady(ctx context.Context, brokers []string, timeout time.Duration, log *zap.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	attempt := 0
	for {
		attempt++

		dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
		conn, err := kafka.DialContext(dialCtx, "tcp", brokers[0])
		dialCancel()

		if err == nil {
			_, cerr := conn.Controller()
			_ = conn.Close()

			if cerr == nil {
				log.Info("kafka: broker is ready", zap.Int("attempts", attempt))
				return nil
			}
			err = cerr
		}

		log.Warn("kafka: broker not ready yet, retrying",
			zap.Int("attempt", attempt), zap.Error(err))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
