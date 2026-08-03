package main

import (
	"context"
	"log"

	"github.com/nhassl3/servicehub-backend/cmd"
	"github.com/nhassl3/servicehub-backend/internal/db"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	repoES "github.com/nhassl3/servicehub-backend/internal/repository/elasticsearch"
	repoPostgres "github.com/nhassl3/servicehub-backend/internal/repository/postgres"
	pkgES "github.com/nhassl3/servicehub-backend/pkg/elasticsearch"
	"github.com/nhassl3/servicehub-backend/pkg/postgres"
)

// main re-index already prepared data from database. This bin collects info about titles, descriptions from rows in
// product table. This needed for FTS (full-text searching) - elasticsearch
func main() {
	ctx := context.Background()
	cfg := cmd.MustLoadConfig()
	logger := cmd.MustLoadLogger(cfg.Log.Level)

	dsn := postgres.DSN(cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode)
	pool, err := postgres.New(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres: %s", err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	productRepo := repoPostgres.NewProductRepo(store)

	esClient, err := pkgES.New(ctx, cfg.ELS.Hosts, cfg.ELS.Username, cfg.ELS.Password)
	if err != nil {
		log.Fatalf("elasticsearch: %s", err)
	}
	defer func() { _ = esClient.Close(context.Background()) }()

	esProductRepo := repoES.NewProductESRepo(esClient, logger)

	if err := esProductRepo.EnsureIndex(ctx); err != nil {
		log.Fatalf("ensure index: %s", err)
	}

	var offset int32
	const batchSize = 100

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

		if err := esProductRepo.BulkIndexProducts(ctx, products); err != nil {
			log.Fatalf("bulk index: %s", err)
		}

		logger.Sugar().Infof("reindexed %d products (offset %d)", len(products), offset)
		offset += batchSize
	}

	logger.Info("reindex complete")
}
