package main

import (
	"context"
	"log"
	"time"

	"github.com/nhassl3/servicehub-backend/cmd"
	"github.com/nhassl3/servicehub-backend/internal/db"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	repoCH "github.com/nhassl3/servicehub-backend/internal/repository/clickhouse"
	repoPostgres "github.com/nhassl3/servicehub-backend/internal/repository/postgres"
	pkgCH "github.com/nhassl3/servicehub-backend/pkg/clickhouse"
)

const batchSize = 32

// main backfills the current catalog state into ClickHouse as business facts.
// Postgres remains the source of truth; this only seeds the analytics
// dashboards (otherwise they start empty until new events arrive).
//
// Coverage: product status + product rating facts are backfilled from the
// products table. Order/moderation/registration history is out of scope — those
// events only exist once the analytics pipeline goes live.
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	cfg := cmd.MustLoadConfig()

	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("postgres: %s", err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	productRepo := repoPostgres.NewProductRepo(store)
	categoryRepo := repoPostgres.NewCategoryRepo(store)

	categories, err := categoryRepo.List(ctx)
	if err != nil {
		log.Fatalf("category_repo: %w", err)
	}

	conn, err := pkgCH.Connect(ctx,
		cfg.Clickhouse.Hosts,
		cfg.Clickhouse.Username,
		cfg.Clickhouse.Database,
		cfg.Clickhouse.Password,
		cfg.Clickhouse.ClientInfo.Product,
		cfg.Clickhouse.ClientInfo.Version,
		cfg.Clickhouse.TLS,
	)
	if err != nil {
		log.Fatalf("clickhouse: %s", err)
	}
	defer func() { _ = conn.Close() }()

	if err := repoCH.EnsureSchema(cfg.Clickhouse, cfg.Migrations.ClickhousePath); err != nil {
		log.Fatalf("clickhouse ensure schema: %s", err)
	}
	repo := repoCH.NewAnalyticsRepo(conn)

	backfillByStatus(ctx, productRepo, categories, repo, "active")
	backfillByStatus(ctx, productRepo, categories, repo, "inactive")
	backfillByStatus(ctx, productRepo, categories, repo, "draft")
}

func backfillByStatus(ctx context.Context, productRepo domain.ProductRepository, categories *domain.ListCategories, repo domain.AnalyticsRepository, status string) {
	var offset int32
	var totalProducts int
	for {
		products, _, err := productRepo.List(ctx, domain.ListProductsParams{
			Status: status,
			Limit:  batchSize,
			Offset: offset,
		})
		if err != nil {
			log.Fatalf("list products: %s", err)
		}
		if len(products) == 0 {
			break
		}

		events := make([]domain.AnalyticsEvent, 0, len(products))
		for _, p := range products {
			var categoryName string
			category := categories.GetCategoryById(p.CategoryID)
			if category != nil {
				categoryName = category.Name
			}
			events = append(events,
				domain.AnalyticsEvent{
					OccurredAt:   p.CreatedAt,
					EventType:    "product_status_changed",
					ProductID:    p.ID,
					SellerID:     p.SellerID,
					CategoryID:   p.CategoryID,
					CategoryName: categoryName,
					Title:        p.Title,
					Status:       p.Status,
					Rating:       p.Rating,
					SalesCount:   p.SalesCount,
					ReviewsCount: p.ReviewsCount,
					Quantity:     -1,
					Total:        p.Price,
				},
			)
		}

		if err := repo.InsertEvents(ctx, events); err != nil {
			log.Fatalf("backfill insert: %s", err)
		}
		totalProducts += len(products)
		log.Printf("backfilled %d products (offset %d)", len(products), offset)
		offset += batchSize
	}
	log.Printf("done: %d products backfilled with status %s", totalProducts, status)
}
