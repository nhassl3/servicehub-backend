package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type TopicSpec struct {
	Name              string
	NumPartitions     int
	ReplicationFactor int
}

// EnsureTopics создаёт топики явно через Admin API, если их ещё нет.
// Устраняет гонку с auto.create.topics.enable=true: при ленивом создании топик
// появляется только в момент первого запроса от ЛЮБОГО клиента (продюсера или
// консьюмера) — и если консьюмер стартовал раньше первого сообщения продюсера,
// он может не восстановиться сам, когда топик появится чуть позже.
// Вызывать один раз при старте — до создания любых Producer/Consumer.
func EnsureTopics(ctx context.Context, brokers []string, specs []TopicSpec, log *zap.Logger) error {
	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial broker: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get controller: %w", err)
	}

	controllerAddr := fmt.Sprintf("%s:%d", controller.Host, controller.Port)
	controllerConn, err := kafka.DialContext(ctx, "tcp", controllerAddr)
	if err != nil {
		return fmt.Errorf("dial controller %s: %w", controllerAddr, err)
	}
	defer controllerConn.Close()

	configs := make([]kafka.TopicConfig, 0, len(specs))
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		configs = append(configs, kafka.TopicConfig{
			Topic:             s.Name,
			NumPartitions:     s.NumPartitions,
			ReplicationFactor: s.ReplicationFactor,
		})
		names = append(names, s.Name)
	}

	if err := controllerConn.CreateTopics(configs...); err != nil {
		if errors.Is(err, kafka.TopicAlreadyExists) {
			log.Debug("kafka: topics already exist, skipping", zap.Strings("topics", names))
			return nil
		}
		return fmt.Errorf("create topics %v: %w", names, err)
	}

	log.Info("kafka: topics ensured", zap.Strings("topics", names))
	return nil
}
