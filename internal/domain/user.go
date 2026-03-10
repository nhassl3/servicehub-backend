package domain

import (
	"context"
	"time"
)

type User struct {
	Username     string    `db:"username"`
	UID          string    `db:"uid"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	FullName     string    `db:"full_name"`
	AvatarURL    string    `db:"avatar_url"`
	Role         string    `db:"role"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type Session struct {
	ID           string    `db:"id"`
	Username     string    `db:"username"`
	RefreshToken string    `db:"refresh_token"`
	UserAgent    string    `db:"user_agent"`
	ClientIP     string    `db:"client_ip"`
	IsBlocked    bool      `db:"is_blocked"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}

type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash string
	FullName     string
}

type CreateSessionParams struct {
	Username     string
	RefreshToken string
	UserAgent    string
	ClientIp     string
	IsBlocked    bool
	ExpiresAt    time.Time
}

type UpdateUserParams struct {
	Username  string
	FullName  string
	AvatarURL string
}

// UpdateUserPasswordParams inputs data
type UpdateUserPasswordParams struct {
	Username    string
	OldPassword string
	NewPassword string
}

//go:generate mockgen -source=user.go -destination=../repository/mock/user_repo_mock.go -package=mockrepo
type UserRepository interface {
	Create(ctx context.Context, params CreateUserParams) (*User, error)
	CreateSession(ctx context.Context, params CreateSessionParams) error
	GetSession(ctx context.Context, username string) (*Session, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUID(ctx context.Context, uid string) (*User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, params UpdateUserParams) (*User, error)
	UpdatePassword(ctx context.Context, params UpdateUserPasswordParams) (*User, error)
}
