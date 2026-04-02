package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Admin struct {
	ID             string    `db:"id"`
	Username       string    `db:"username"`
	DisplayName    string    `db:"display_name"`
	TotalModerates int32     `db:"total_moderation"`
	LevelRights    int32     `db:"level_rights"`
	AvatarURL      string    `db:"avatar_url"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// MarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// marshalling Admin structure to the []byte code (JSON)
func (a *Admin) MarshalBinary() ([]byte, error) {
	return json.Marshal(a)
}

// UnmarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// unmarshalling source data and convert to Admin structure
func (a *Admin) UnmarshalBinary(data []byte) error {
	if a == nil {
		return ErrRedisNotFound
	}
	return json.Unmarshal(data, a)
}

type CreateAdminParams struct {
	Username    string
	DisplayName string
	LevelRights int32
}

type GetAdminProfileParams struct {
	Username *string
	AdminId  *string
}

type UpdateAdminsProfileParams struct {
	Username       string
	DisplayName    *string
	LevelRights    *int32
	TotalModerates *int32
	AvatarURL      *string
}

type UploadCategoryAvatar struct {
	Slug        string
	FileData    []byte
	ContentType string
}

type UploadAdminAvatar struct {
	Username    string
	FileData    []byte
	ContentType string
}

type IncreaseTotalModeratesParams struct {
	Username       string
	TotalModerates int32
}

//go:generate mockgen -source=seller.go -destination=../repository/mock/seller_repo_mock.go -package=mockrepo
type AdminRepository interface {
	CreateAdmin(ctx context.Context, params CreateAdminParams) (*Admin, error)
	GetAdmin(ctx context.Context, params GetAdminProfileParams) (*Admin, error)
	UpdateAdminProfile(ctx context.Context, params UpdateAdminsProfileParams) (*Admin, error)
	UploadCategoryAvatar(ctx context.Context, params UploadCategoryAvatar) (*Category, error)
	IncreaseTotalModerates(ctx context.Context, params IncreaseTotalModeratesParams) error
	ExistsAdminByUsername(ctx context.Context, username string) (bool, error)
}

type AdminRedis interface {
	Profile(ctx context.Context, username string) (*Admin, error)
	SetProfile(ctx context.Context, user *Admin) error
}
