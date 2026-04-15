package redis

import (
	"context"
	"errors"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/redis/go-redis/v9"
)

const categoriesPrefix = "servicehub:categories"

type CategoriesRedis struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCategoryRedis(client *redis.Client, ttl time.Duration) *CategoriesRedis {
	return &CategoriesRedis{
		client: client,
		ttl:    ttl,
	}
}

// Categories get categories from Redis
// TTL: 1 hour
func (c *CategoriesRedis) Categories(ctx context.Context) (*domain.ListCategories, error) {
	// If the capacity is greater than the desired value, the capacity value will increase.
	categories := make(domain.ListCategories, 0, 10) // len 0, cap: 10
	if err := c.client.Get(ctx, categoriesPrefix).Scan(&categories); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrRedisNotFound
		}
		return nil, err
	}
	return &categories, nil
}

// SetCategories set categories to Redis database
// in parameters you should give categories
func (c *CategoriesRedis) SetCategories(ctx context.Context, categories *domain.ListCategories) error {
	return c.client.Set(ctx, categoriesPrefix, categories, c.ttl).Err()
}
