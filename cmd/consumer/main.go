package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	"go.uber.org/zap"

	"github.com/nhassl3/servicehub-backend/cmd"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	pkgkafka "github.com/nhassl3/servicehub-backend/pkg/kafka"
)

func main() {
	cfg := cmd.MustLoadConfig()
	log := cmd.MustLoadLogger(cfg.Log.Level)
	defer func(log *zap.Logger) {
		_ = log.Sync()
	}(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var (
		smtpClient mailer.Notifier
		err        error
	)
	if cfg.Environment == "local" {
		smtpClient = mailer.NewNoopNotifier(log)
	} else {
		smtpClient, err = mailer.NewSMTPMailer(
			cfg.SMTP.Host, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.FromEmail, cfg.SMTP.Port, log,
		)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	dlqProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.NotificationsDLQ, log)
	defer dlqProducer.Close()

	orderConsumer := pkgkafka.NewConsumer(
		cfg.Kafka.Brokers, cfg.Kafka.Topics.OrderEvents, cfg.Kafka.GroupID+"-order-events", log,
	).WithDLQ(dlqProducer)
	txConsumer := pkgkafka.NewConsumer(
		cfg.Kafka.Brokers, cfg.Kafka.Topics.TransactionEvents, cfg.Kafka.GroupID+"-transaction-events", log,
	)
	defer orderConsumer.Close()
	defer txConsumer.Close()

	orderNotifConsumer := kafka.NewNotificationConsumer(orderConsumer, smtpClient, log)
	txNotifConsumer := kafka.NewNotificationConsumer(txConsumer, smtpClient, log)

	var wg sync.WaitGroup
	wg.Add(2) // len(cfg.Kafka.Topic)

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

	log.Info("kafka consumer service started", zap.String("env", cfg.Environment), zap.String("Mode", cfg.Log.Level))
	wg.Wait()
	log.Info("kafka consumer service stopped gracefully")
}
