package domain

import (
	"context"
	"encoding/json"
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

// MarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// marshalling User structure to the []byte code (JSON)
func (u *User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

// UnmarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// unmarshalling source data and convert to User structure
func (u *User) UnmarshalBinary(data []byte) error {
	if u == nil {
		return ErrRedisNotFound
	}
	return json.Unmarshal(data, u)
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

// MarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// marshalling Session structure to the []byte code (JSON)
func (u *Session) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

// UnmarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// unmarshalling source data and convert to Session structure
func (u *Session) UnmarshalBinary(data []byte) error {
	if u == nil {
		return ErrRedisNotFound
	}
	return json.Unmarshal(data, u)
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

type UploadAvatarParams struct {
	Username    string
	FileData    []byte
	ContentType string
}

type RequestResetPasswordParams struct {
	Username,
	Email *string
}

type ResetPasswordState struct {
	Email,
	Code string
}

// MarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// marshalling User structure to the []byte code (JSON)
func (c *ResetPasswordState) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// unmarshalling source data and convert to User structure
func (c *ResetPasswordState) UnmarshalBinary(data []byte) error {
	if c == nil {
		return ErrRedisNotFound
	}
	return json.Unmarshal(data, c)
}

func NewCode(email, code string) *ResetPasswordState {
	return &ResetPasswordState{
		Email: email,
		Code:  code,
	}
}

type VerifyEmailAccount struct {
	Email,
	Username *string
}

//go:generate mockgen -source=user.go -destination=../repository/mock/user_repo_mock.go -package=mockrepo
type UserRepository interface {
	Create(ctx context.Context, params CreateUserParams) (*User, error)
	CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error)
	GetSession(ctx context.Context, refreshToken string) (*Session, error)
	GetSessionByUsername(ctx context.Context, username string) (*Session, error)
	DeleteSession(ctx context.Context, username string) error
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUID(ctx context.Context, uid string) (*User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, params UpdateUserParams) (*User, error)
	UpdatePassword(ctx context.Context, params UpdateUserPasswordParams) (*User, error)
	VerifyEmail(ctx context.Context, params VerifyEmailAccount) (*User, error)
}

type UserRedis interface {
	Profile(ctx context.Context, username string) (*User, error)
	Session(ctx context.Context, username string) (*Session, error)
	AuthBlock(ctx context.Context, clientIP string) (bool, float64, error)
	SetProfile(ctx context.Context, user *User) error
	SetSession(ctx context.Context, session *Session) error
	SetAuthBlock(ctx context.Context, clientIP string) error
	DelProfile(ctx context.Context, username string) error
	DelSession(ctx context.Context, username string) error
	Code(ctx context.Context, enterKeyCode string, operationId string) (*ResetPasswordState, error)
	SetCode(ctx context.Context, enterKeyCode, operationId string, code *ResetPasswordState) error
	Verified(ctx context.Context, entryCode, token string) (string, error)
	SetVerified(ctx context.Context, entryCode, token, email string) error
	DelVerified(ctx context.Context, entryCode, token string) error
	DelCode(ctx context.Context, enterKeyCode, operationId string) error
}
