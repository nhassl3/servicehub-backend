package redis

import (
	"context"
	"errors"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	sessionPrefix   = "session:"
	profilePrefix   = "profile:"
	authBlockPrefix = "auth:block:"
)

type UserRedis struct {
	client       *redis.Client
	profileTTL   time.Duration
	authBlockTTL time.Duration
}

func NewUserRedis(client *redis.Client, profileTTL, authBlockTTL time.Duration) *UserRedis {
	return &UserRedis{
		client:       client,
		profileTTL:   profileTTL,
		authBlockTTL: authBlockTTL,
	}
}

func (u *UserRedis) Profile(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User

	if err := u.client.Get(ctx, profilePrefix+username).Scan(&user); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrRedisNotFound
		}
		return nil, err
	}

	return &user, nil
}

// Session get record from Redis about session.
// Standard ttl - 168h that equal auth refresh token ttl.
// When user first launch the WEB session store in database and Redis.
// But when Redis have a key with this session he returns a message with
// session and ttl equal ttl of refreshToken that has been given early
func (u *UserRedis) Session(ctx context.Context, username string) (*domain.Session, error) {
	var session domain.Session

	if err := u.client.Get(ctx, sessionPrefix+username).Scan(&session); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrRedisNotFound
		}
		return nil, err
	}

	return &session, nil
}

// AuthBlock this method calls when system need to check if user call Register or Login from one IP one more time.
// If yes - block this endpoints in redis store on TTL time that store in config file.
func (u *UserRedis) AuthBlock(ctx context.Context, clientIP string) (bool, float64, error) {
	ok, err := u.client.Get(ctx, authBlockPrefix+clientIP).Bool()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, -1, nil
		}
		return false, -1, err
	}
	ttl := u.client.TTL(ctx, authBlockPrefix+clientIP).Val()
	return ok, ttl.Seconds(), nil
}

// SetProfile create record in Redis for user profile on 15 minutes
func (u *UserRedis) SetProfile(ctx context.Context, user *domain.User) error {
	return u.client.Set(ctx, profilePrefix+user.Username, user, u.profileTTL).Err()
}

// SetSession write a record in Redis database with a ttl that equal left of refreshToken
// ExpiresAt - Now = ttl of record Redis session
func (u *UserRedis) SetSession(ctx context.Context, session *domain.Session) error {
	return u.client.Set(ctx, sessionPrefix+session.Username, session, time.Until(session.ExpiresAt)).Err()
}

// SetAuthBlock this method calls when user successfully log in or sign up.
// That's needed for brute force protection
func (u *UserRedis) SetAuthBlock(ctx context.Context, clientIP string) error {
	return u.client.Set(ctx, authBlockPrefix+clientIP, true, u.authBlockTTL).Err()
}

func (u *UserRedis) DelProfile(ctx context.Context, username string) error {
	return u.client.Del(ctx, profilePrefix+username).Err()
}

func (u *UserRedis) DelSession(ctx context.Context, username string) error {
	return u.client.Del(ctx, sessionPrefix+username).Err()
}
