package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/nhassl3/servicehub-backend/internal/config"
	"github.com/nhassl3/servicehub-backend/internal/db"
	repoClickhouse "github.com/nhassl3/servicehub-backend/internal/repository/clickhouse"
	repoES "github.com/nhassl3/servicehub-backend/internal/repository/elasticsearch"
	repoPostgres "github.com/nhassl3/servicehub-backend/internal/repository/postgres"
	repoRedis "github.com/nhassl3/servicehub-backend/internal/repository/redis"
	"github.com/nhassl3/servicehub-backend/internal/service"
	serviceKafka "github.com/nhassl3/servicehub-backend/internal/service/kafka"
	transportGRPC "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	transportHTTP "github.com/nhassl3/servicehub-backend/internal/transport/http"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/nhassl3/servicehub-backend/pkg/clickhouse"
	pkgES "github.com/nhassl3/servicehub-backend/pkg/elasticsearch"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	pkgRedis "github.com/nhassl3/servicehub-backend/pkg/redis"
	minio "github.com/nhassl3/servicehub-backend/pkg/storage"
	"go.uber.org/zap"
)

// Run bootstraps and starts the application.
func Run(cfg *config.Config, log *zap.Logger) error {
	ctx := context.Background()

	// ─── Database ─────────────────────────────────────────────────────────────
	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("connected to PostgresSQL")

	// ─── Migrations ───────────────────────────────────────────────────────────
	if cfg.Environment == "local" {
		if err := repoPostgres.EnsureSchema(cfg.DB, cfg.Migrations.PostgresPath); err != nil {
			return fmt.Errorf("app: ensure postgres schema: %w", err)
		}
		if err := repoClickhouse.EnsureSchema(cfg.Clickhouse, cfg.Migrations.ClickhousePath); err != nil {
			return fmt.Errorf("app: ensure clickhouse schema: %w", err)
		}
	}
	log.Info("Successfully applied all migration (Clickhouse and Postgres)")

	// ─── SQLC Store ─────────────────────────────────────────────────────────
	store := db.NewStore(pool)

	// ─── Redis ────────────────────────────────────────────────────────────────
	redisClient, err := pkgRedis.New(ctx, cfg.Redis.Addr, cfg.Redis.Username, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return fmt.Errorf("app: connect redis: %w", err)
	}
	defer func() { _ = redisClient.Close() }()
	log.Info("connected to Redis")

	redisProductsClient, err := pkgRedis.New(ctx, cfg.Redis.Addr, cfg.Redis.Username, cfg.Redis.Password, cfg.Redis.DB+1)
	if err != nil {
		return fmt.Errorf("app: connect redis products: %w", err)
	}
	defer func() { _ = redisProductsClient.Close() }()
	log.Info("connected to RedisProducts")

	// RedisClient store tokens, sessions, profiles
	tokenBlacklist := repoRedis.NewTokenBlacklist(redisClient)
	userRedis := repoRedis.NewUserRedis(
		redisClient, cfg.Redis.TTL.User, cfg.Redis.TTL.AuthBlock, cfg.Redis.TTL.Code, cfg.Redis.TTL.ResetPassword,
	)
	adminRedis := repoRedis.NewAdminRedis(redisClient, cfg.Redis.TTL.Product, cfg.Redis.TTL.Claim)

	// RedisProducts store categories and products closed on a page a few moments later
	categoriesRedis := repoRedis.NewCategoryRedis(redisProductsClient, cfg.Redis.TTL.Categories)

	// ─── Kafka ────────────────────────────────────────────────────────────────

	// Топики создаются явно ДО подписки консьюмеров — иначе при auto.create.topics.enable=true
	// consumer group может попытаться присоединиться к топику, которого физически ещё нет
	// (он создастся позже, лениво, при первой публикации из backend), и не восстановится сама.
	topics := make([]kafka.TopicSpec, 0, len(cfg.Kafka.Topics.Events))
	for _, e := range cfg.Kafka.Topics.Events {
		topics = append(topics, kafka.TopicSpec{Name: e, NumPartitions: 3, ReplicationFactor: 1})
	}
	if err = kafka.EnsureTopics(ctx, cfg.Kafka.Brokers, topics, log); err != nil {
		log.Fatal("kafka: failed to ensure topics exist, exiting for restart", zap.Error(err))
	}

	producers := make([]*kafka.Producer, 0, len(cfg.Kafka.Topics.Events)+1)
	eventProducers := make(map[string]*kafka.Producer, len(cfg.Kafka.Topics.Events))
	for _, p := range cfg.Kafka.Topics.Events {
		kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, p, log)
		eventProducers[p] = kafkaProducer
		producers = append(producers, kafkaProducer)
	}
	producers = append(producers, kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Notifications, log))

	eventPublisher := serviceKafka.NewEventPublisher(eventProducers)

	// ─── Clickhouse ───────────────────────────────────────────────────────────
	conn, err := clickhouse.Connect(ctx,
		cfg.Clickhouse.Hosts,
		cfg.Clickhouse.Username,
		cfg.Clickhouse.Database,
		cfg.Clickhouse.Password,
		cfg.Clickhouse.ClientInfo.Product,
		cfg.Clickhouse.ClientInfo.Version,
		cfg.Clickhouse.TLS,
	)
	if err != nil {
		return fmt.Errorf("app: connect clickhouse: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	log.Info("connected to ClickHouse")

	// ─── MinIO ────────────────────────────────────────────────────────────────
	minIOClient, err := minio.NewMinIO(
		ctx,
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		"",
		cfg.MinIO.Bucket,
		cfg.MinIO.UseSSL,
	)
	if err != nil {
		return fmt.Errorf("minio.NewMinIO: failed to initialize minio client: %w", err)
	}
	log.Info("connected to MinIO")

	// ─── Elasticsearch ─────────────────────────────────────────────────────────
	esClient, err := pkgES.New(ctx, cfg.ELS.Hosts, cfg.ELS.Username, cfg.ELS.Password)
	if err != nil {
		return fmt.Errorf("app: connect elasticsearch: %w", err)
	}
	defer func() { _ = esClient.Close(ctx) }()
	log.Info("connected to Elasticsearch")

	// ─── Token managers ───────────────────────────────────────────────────────
	accessMaker, err := auth.NewPasetoMaker(cfg.Auth.PasetoKey, cfg.Auth.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("app: create paseto access maker: %w", err)
	}
	accessManager := auth.NewBlacklistedTokenManager(accessMaker, tokenBlacklist)

	refreshManager, err := auth.NewPasetoMaker(cfg.Auth.PasetoKey, cfg.Auth.RefreshTokenTTL)
	if err != nil {
		return fmt.Errorf("app: create paseto refresh maker: %w", err)
	}

	// ─── Mailer (SMTP Client) ─────────────────────────────────────────────────
	var smtpClient mailer.Notifier
	if cfg.Environment == "local" {
		smtpClient = mailer.NewNoopNotifier(log)
	} else {
		smtpClient, err = mailer.NewSMTPMailer(
			cfg.SMTP.Host, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.FromEmail, cfg.SMTP.Port, log,
		)
		if err != nil {
			return fmt.Errorf("app: create smtp client: %w", err)
		}
	}

	// ─── Repositories ─────────────────────────────────────────────────────────
	userRepo := repoPostgres.NewUserRepo(store)
	categoryRepo := repoPostgres.NewCategoryRepo(store)
	productRepo := repoPostgres.NewProductRepo(store)
	sellerRepo := repoPostgres.NewSellerRepo(store)
	cartRepo := repoPostgres.NewCartRepo(store)
	orderRepo := repoPostgres.NewOrderRepo(store)
	reviewRepo := repoPostgres.NewReviewRepo(store)
	wishlistRepo := repoPostgres.NewWishlistRepo(store)
	balanceRepo := repoPostgres.NewBalanceRepo(store)
	adminRepo := repoPostgres.NewAdminRepo(store)
	moderationRepo := repoPostgres.NewModerationRepo(store)
	notificationRepo := repoPostgres.NewNotificationRepository(store)
	esProductRepo := repoES.NewProductESRepo(esClient, log) // elasticsearch
	if err := esProductRepo.EnsureIndex(ctx); err != nil {
		return fmt.Errorf("app: ensure elasticsearch index: %w", err)
	}
	analyticsRepo := repoClickhouse.NewAnalyticsRepo(conn)

	// ─── Services ─────────────────────────────────────────────────────────────
	svcs := &transportGRPC.Services{
		Admin:        service.NewAdminService(adminRepo, minIOClient, adminRedis),
		Auth:         service.NewAuthService(userRepo, userRedis, accessManager, refreshManager, tokenBlacklist, smtpClient, eventPublisher, log),
		User:         service.NewUserService(userRepo, minIOClient, userRedis),
		Category:     service.NewCategoryService(categoryRepo, categoriesRedis, minIOClient),
		Product:      service.NewProductService(productRepo, esProductRepo, sellerRepo, eventPublisher, log),
		Cart:         service.NewCartService(cartRepo),
		Order:        service.NewOrderService(orderRepo, productRepo, eventPublisher, userRedis, log),
		Review:       service.NewReviewService(reviewRepo, productRepo, eventPublisher, log),
		Wishlist:     service.NewWishlistService(wishlistRepo),
		Seller:       service.NewSellerService(sellerRepo, minIOClient, userRedis),
		Balance:      service.NewBalanceService(balanceRepo, eventPublisher, userRedis, log),
		Moderation:   service.NewModerationService(moderationRepo, adminRepo, productRepo, adminRedis, adminRedis, eventPublisher, log),
		Notification: service.NewNotificationService(userRedis, userRepo, notificationRepo),
		Analytics:    service.NewAnalyticsService(analyticsRepo, adminRepo, adminRedis),
	}

	// ─── gRPC Server ──────────────────────────────────────────────────────────
	grpcServer := transportGRPC.NewServer(svcs, accessManager, log)

	// ─── Start servers ────────────────────────────────────────────────────────
	errCh := make(chan error, 2)

	// gRPC server
	go func() {
		if err := grpcServer.Start(cfg.Server.GRPCPort); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	// gateway gRPC server (HTTP)
	go func() {
		if err := grpcServer.StartGateway(ctx, "localhost"+cfg.Server.GRPCPort, cfg.Server.HTTPPort); err != nil {
			errCh <- fmt.Errorf("error http gateway: %w", err)
		}
	}()

	log.Info("ServiceHub started",
		zap.String("grpc_port", cfg.Server.GRPCPort),
		zap.String("http_port", cfg.Server.HTTPPort),
		zap.String("env", cfg.Environment),
	)

	// ─── pprof (local/dev only) ───────────────────────────────────────────────
	// Runtime profiling endpoint bound to 127.0.0.1. Never exposed in prod:
	// the server goroutine is not spawned at all for non-local environments.
	if cfg.Environment == "local" || cfg.Environment == "dev" {
		pprofServer := transportHTTP.NewPprofServer(cfg.Server.PPROFPort)
		go func() {
			log.Info("pprof server listening", zap.String("addr", pprofServer.Addr))
			if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("pprof server failed", zap.Error(err))
			}
		}()
	}

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
		log.Info("shutting down gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*1e9)
		defer cancel()

		var kafkaProducerError error
		for _, e := range producers {
			kafkaProducerError = errors.Join(e.Close())
		}
		if kafkaProducerError != nil {
			return kafkaProducerError
		}

		grpcServer.Shutdown(shutdownCtx)
		return nil
	}
}
