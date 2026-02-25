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
	"github.com/nhassl3/servicehub/internal/config"
	"github.com/nhassl3/servicehub/internal/db"
	repoPostgres "github.com/nhassl3/servicehub/internal/repository/postgres"
	"github.com/nhassl3/servicehub/internal/service"
	transportGRPC "github.com/nhassl3/servicehub/internal/transport/grpc"
	"github.com/nhassl3/servicehub/pkg/auth"
	"github.com/nhassl3/servicehub/pkg/logger"
	"github.com/nhassl3/servicehub/pkg/postgres"
	"go.uber.org/zap"
)

// Run bootstraps and starts the application.
func Run(cfg *config.Config) error {
	// ─── Logger ───────────────────────────────────────────────────────────────
	log, err := logger.NewZapLogger(cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("app: init logger: %w", err)
	}
	defer func(log *zap.Logger) {
		_ = log.Sync()
	}(log) //nolint:errcheck

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

	// ─── Token managers ───────────────────────────────────────────────────────
	accessManager, err := auth.NewPasetoMaker(cfg.Auth.PasetoKey, cfg.Auth.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("app: create paseto access maker: %w", err)
	}
	refreshManager, err := auth.NewPasetoMaker(cfg.Auth.PasetoKey, cfg.Auth.RefreshTokenTTL)
	if err != nil {
		return fmt.Errorf("app: create paseto refresh maker: %w", err)
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

	// ─── Services ─────────────────────────────────────────────────────────────
	svcs := &transportGRPC.Services{
		Auth:     service.NewAuthService(userRepo, accessManager, refreshManager),
		User:     service.NewUserService(userRepo),
		Category: service.NewCategoryService(categoryRepo),
		Product:  service.NewProductService(productRepo, sellerRepo),
		Cart:     service.NewCartService(cartRepo),
		Order:    service.NewOrderService(orderRepo),
		Review:   service.NewReviewService(reviewRepo),
		Wishlist: service.NewWishlistService(wishlistRepo),
		Seller:   service.NewSellerService(sellerRepo),
		Balance:  service.NewBalanceService(balanceRepo),
	}

	// ─── gRPC Server ──────────────────────────────────────────────────────────
	grpcServer := transportGRPC.NewServer(svcs, accessManager, log)

	// ─── Start servers ────────────────────────────────────────────────────────
	errCh := make(chan error, 2)

	go func() {
		if err := grpcServer.Start(cfg.Server.GRPCPort); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

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
		return grpcServer.Shutdown(shutdownCtx)
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
