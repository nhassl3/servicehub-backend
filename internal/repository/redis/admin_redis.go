package redis

import (
	"context"
	"errors"
	"time"

	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/redis/go-redis/v9"
)

const adminKeyPrefix = "admin:"

type AdminRedis struct {
	client     *redis.Client
	profileTTL time.Duration
}

func NewAdminRedis(client *redis.Client, profileTTL time.Duration) *AdminRedis {
	return &AdminRedis{
		client:     client,
		profileTTL: profileTTL,
	}
}

func (a *AdminRedis) Profile(ctx context.Context, username string) (*domain.Admin, error) {
	var admin domain.Admin

	if err := a.client.Get(ctx, adminKeyPrefix+username).Scan(&admin); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrRedisNotFound
		}
		return nil, err
	}

	return &admin, nil
}

func (a *AdminRedis) SetProfile(ctx context.Context, admin *domain.Admin) error {
	return a.client.Set(ctx, adminKeyPrefix+admin.Username, admin, a.profileTTL).Err()
}
