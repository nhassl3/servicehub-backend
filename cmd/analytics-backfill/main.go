package main

import (
	"context"
	"log"

	"github.com/nhassl3/servicehub-backend/cmd"
	"github.com/nhassl3/servicehub-backend/internal/db"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	repoCH "github.com/nhassl3/servicehub-backend/internal/repository/clickhouse"
	repoPostgres "github.com/nhassl3/servicehub-backend/internal/repository/postgres"
	pkgCH "github.com/nhassl3/servicehub-backend/pkg/clickhouse"
)

// main backfills the current catalog state into ClickHouse as business facts.
// Postgres remains the source of truth; this only seeds the analytics
// dashboards (otherwise they start empty until new events arrive).
//
// Coverage: product status + product rating facts are backfilled from the
// products table. Order/moderation/registration history is out of scope — those
// events only exist once the analytics pipeline goes live.
func main() {
	ctx := context.Background()
	cfg := cmd.MustLoadConfig()

	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("postgres: %s", err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	productRepo := repoPostgres.NewProductRepo(store)

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

	const batchSize = 32
	var offset int32
	var totalProducts int

	for {
		products, _, err := productRepo.List(ctx, domain.ListProductsParams{
			Limit:  batchSize,
			Offset: offset,
		})
		if err != nil {
			log.Fatalf("list products: %s", err)
		}
		if len(products) == 0 {
			break
		}

		events := make([]domain.AnalyticsEvent, 0, len(products)*2)
		for _, p := range products {
			events = append(events,
				domain.AnalyticsEvent{
					OccurredAt:   p.CreatedAt,
					EventType:    "product_status_changed",
					ProductID:    p.ID,
					SellerID:     p.SellerID,
					CategoryID:   p.CategoryID,
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
	log.Printf("done: %d products backfilled", totalProducts)
}
