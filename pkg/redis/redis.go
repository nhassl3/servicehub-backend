package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// New creates a Redis client and verifies the connection with a PING.
func New(ctx context.Context, addr, username, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return client, nil
}
