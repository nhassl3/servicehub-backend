package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	elsRepo "github.com/nhassl3/servicehub-backend/internal/repository/elasticsearch"
	elsPkg "github.com/nhassl3/servicehub-backend/pkg/elasticsearch"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	"go.uber.org/zap"

	repoCH "github.com/nhassl3/servicehub-backend/internal/repository/clickhouse"
	pkgCH "github.com/nhassl3/servicehub-backend/pkg/clickhouse"

	"github.com/nhassl3/servicehub-backend/cmd"
	transportKafka "github.com/nhassl3/servicehub-backend/internal/transport/kafka"
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

	// Если брокер не подтвердил готовность за 90с — падаем. С restart-политикой
	// в docker-compose Docker поднимет контейнер заново; к этому моменту Kafka,
	// скорее всего, уже полностью прошла холодную инициализацию (в частности,
	// создание __consumer_offsets), и повторная попытка пройдёт быстро.
	if err := pkgkafka.WaitReady(ctx, cfg.Kafka.Brokers, 90*time.Second, log); err != nil {
		log.Fatal("kafka: broker did not become ready in time, exiting for restart", zap.Error(err))
	}

	// Топики создаются явно ДО подписки консьюмеров — иначе при auto.create.topics.enable=true
	// consumer group может попытаться присоединиться к топику, которого физически ещё нет
	// (он создастся позже, лениво, при первой публикации из backend), и не восстановится сама.
	topics := make([]kafka.TopicSpec, 0, len(cfg.Kafka.Topics.Events))
	for _, e := range cfg.Kafka.Topics.Events {
		topics = append(topics, kafka.TopicSpec{Name: e, NumPartitions: 3, ReplicationFactor: 1})
	}
	if err := kafka.EnsureTopics(ctx, cfg.Kafka.Brokers, topics, log); err != nil {
		log.Fatal("kafka: failed to ensure topics exist, exiting for restart", zap.Error(err))
	}

	// SMTP connect
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

	// Elasticsearch connect
	elsClient, err := elsPkg.New(ctx, cfg.ELS.Hosts, cfg.ELS.Username, cfg.ELS.Password)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func(elsClient *elasticsearch.Client) {
		_ = elsClient.Close(ctx)
	}(elsClient)

	elsRepository := elsRepo.NewProductESRepo(elsClient, log)

	// ClickHouse connect — analytics consumer writes OLAP facts here.
	chConn, err := pkgCH.Connect(ctx,
		cfg.Clickhouse.Hosts,
		cfg.Clickhouse.Username,
		cfg.Clickhouse.Database,
		cfg.Clickhouse.Password,
		cfg.Clickhouse.ClientInfo.Product,
		cfg.Clickhouse.ClientInfo.Version,
		cfg.Clickhouse.TLS,
	)
	if err != nil {
		log.Fatal("clickhouse: connect failed", zap.Error(err))
	}
	defer func() {
		_ = chConn.Close()
	}()
	if cfg.Log.Level == "local" {
		if err := repoCH.EnsureSchema(cfg.Clickhouse, cfg.Migrations.ClickhousePath); err != nil {
			log.Fatal("clickhouse: ensure schema failed", zap.Error(err))
		}
	}

	analyticsRepo := repoCH.NewAnalyticsRepo(chConn)

	dlqProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.NotificationsDLQ, log)
	defer func() {
		_ = dlqProducer.Close()
	}()

	consumers := make(map[domain.Topic]*pkgkafka.Consumer, len(cfg.Kafka.Topics.Events))
	for _, c := range cfg.Kafka.Topics.Events {
		var consumer *pkgkafka.Consumer
		if slices.Contains(domain.DLQTopics, domain.Topic(c)) {
			consumer = pkgkafka.NewConsumer(
				cfg.Kafka.Brokers, c, fmt.Sprintf("%s-%s", cfg.Kafka.GroupID, c), log,
			).WithDLQ(dlqProducer)
		} else {
			consumer = pkgkafka.NewConsumer(
				cfg.Kafka.Brokers, c, fmt.Sprintf("%s-%s", cfg.Kafka.GroupID, c), log,
			)
		}
		consumers[domain.Topic(c)] = consumer
	}

	handlers := make([]transportKafka.ConsumerHandler, 0, len(consumers))
	for t, consumer := range consumers {
		switch t {
		case domain.TopicOrderEvent, domain.TopicTransactionEvent:
			handlers = append(handlers, transportKafka.NewNotificationConsumer(consumer, smtpClient, log))
		case domain.TopicProductEvent:
			handlers = append(handlers, transportKafka.NewProductConsumer(consumer, elsRepository, log))
		case domain.TopicAnalyticsEvent:
			handlers = append(handlers, transportKafka.NewAnalyticsConsumer(consumer, analyticsRepo, log))
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(handlers))

	for _, h := range handlers {
		go func() {
			defer wg.Done()
			if err = h.Run(ctx); err != nil {
				log.Error("kafka: consumer stopped", zap.Error(err))
			}
		}()
	}

	log.Info("kafka consumer service started", zap.String("env", cfg.Environment), zap.String("Mode", cfg.Log.Level))
	log.Info("───────────────────────────────────────────────────────────────────────────────────────────────────")
	wg.Wait()
	var consumerErrors error
	for _, consumer := range consumers {
		consumerErrors = errors.Join(consumer.Close())
	}
	if consumerErrors != nil {
		log.Fatal(consumerErrors.Error())
	}
	log.Info("kafka consumer service stopped gracefully")
}
