package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/nhassl3/servicehub-backend/internal/config"
	"github.com/nhassl3/servicehub-backend/internal/db"
	repoPostgres "github.com/nhassl3/servicehub-backend/internal/repository/postgres"
	repoRedis "github.com/nhassl3/servicehub-backend/internal/repository/redis"
	"github.com/nhassl3/servicehub-backend/internal/service"
	serviceKafka "github.com/nhassl3/servicehub-backend/internal/service/kafka"
	transportGRPC "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/nhassl3/servicehub-backend/pkg/kafka"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	"github.com/nhassl3/servicehub-backend/pkg/postgres"
	pkgRedis "github.com/nhassl3/servicehub-backend/pkg/redis"
	minio "github.com/nhassl3/servicehub-backend/pkg/storage"
	"go.uber.org/zap"
)

// Run bootstraps and starts the application.
func Run(cfg *config.Config, log *zap.Logger) error {
	// ─── Database ─────────────────────────────────────────────────────────────
	ctx := context.Background()
	dsn := postgres.DSN(cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode)

	pool, err := postgres.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("app: connect postgres: %w", err)
	}
	defer pool.Close()
	log.Info("connected to PostgresSQL")

	// ─── Migrations ───────────────────────────────────────────────────────────
	if cfg.Environment == "local" {
		if err := runMigrations(dsn, log); err != nil {
			return fmt.Errorf("app: run migrations: %w", err)
		}
	}

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

	producers := make([]*kafka.Producer, 0, 3)
	kafkaOrderProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.OrderEvents, log)
	kafkaTransactionProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.TransactionEvents, log)
	kafkaNotificationsProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Notifications, log)
	producers = append(producers, kafkaOrderProducer, kafkaTransactionProducer, kafkaNotificationsProducer)

	eventPublisher := serviceKafka.NewEventPublisher(kafkaOrderProducer, kafkaTransactionProducer)

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

	// ─── Services ─────────────────────────────────────────────────────────────
	svcs := &transportGRPC.Services{
		Admin:        service.NewAdminService(adminRepo, minIOClient, adminRedis),
		Auth:         service.NewAuthService(userRepo, userRedis, accessManager, refreshManager, tokenBlacklist, smtpClient),
		User:         service.NewUserService(userRepo, minIOClient, userRedis),
		Category:     service.NewCategoryService(categoryRepo, categoriesRedis, minIOClient),
		Product:      service.NewProductService(productRepo, sellerRepo),
		Cart:         service.NewCartService(cartRepo),
		Order:        service.NewOrderService(orderRepo, eventPublisher, userRedis, log),
		Review:       service.NewReviewService(reviewRepo),
		Wishlist:     service.NewWishlistService(wishlistRepo),
		Seller:       service.NewSellerService(sellerRepo, minIOClient),
		Balance:      service.NewBalanceService(balanceRepo),
		Moderation:   service.NewModerationService(moderationRepo, adminRepo, productRepo, adminRedis, adminRedis),
		Notification: service.NewNotificationService(userRedis, userRepo, notificationRepo),
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

func runMigrations(dsn string, log *zap.Logger) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("create migrate: %w", err)
	}
	defer func(m *migrate.Migrate) {
		_, _ = m.Close()
	}(m) //nolint:errcheck

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	log.Info("migrations applied")
	return nil
}
