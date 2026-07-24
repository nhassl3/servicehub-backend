package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"

	"github.com/nhassl3/servicehub-backend/cmd"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	pkgkafka "github.com/nhassl3/servicehub-backend/pkg/kafka"
)

// stdoutNotifier — временная заглушка. Замените на реальный email/push/websocket сервис.
type stdoutNotifier struct{ log *zap.Logger }

func (n *stdoutNotifier) Notify(_ context.Context, userID int64, title, body string) error {
	n.log.Info("notification", zap.Int64("user_id", userID), zap.String("title", title), zap.String("body", body))
	return nil
}

func main() {
	cfg := cmd.MustLoadConfig()
	log := cmd.MustLoadLogger(cfg.Log.Level)
	defer func(log *zap.Logger) {
		err := log.Sync()
		if err != nil {
			log.Fatal("logger sync error", zap.Error(err))
		}
	}(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	notifier := &stdoutNotifier{log: log}

	orderConsumer := pkgkafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topics.OrderEvents, cfg.Kafka.GroupID, log)
	txConsumer := pkgkafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topics.TransactionEvents, cfg.Kafka.GroupID, log)
	defer orderConsumer.Close()
	defer txConsumer.Close()

	orderNotifConsumer := kafka.NewNotificationConsumer(orderConsumer, notifier, log)
	txNotifConsumer := kafka.NewNotificationConsumer(txConsumer, notifier, log)

	var wg sync.WaitGroup
	wg.Add(2) // len(cfg.Kafka.

	go func() {
		defer wg.Done()
		if err := orderNotifConsumer.Run(ctx); err != nil {
			log.Error("order notification consumer stopped", zap.Error(err))
		}
	}()

	go func() {
		defer wg.Done()
		if err := txNotifConsumer.Run(ctx); err != nil {
			log.Error("transaction notification consumer stopped", zap.Error(err))
		}
	}()

	log.Info("kafka consumer service started")
	wg.Wait()
	log.Info("kafka consumer service stopped gracefully")
}
