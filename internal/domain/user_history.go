package domain

import (
	"context"
	"time"
)

type Action int

const (
	Unspecified Action = iota
	Created
	Updated
	Deleted
	Restored
	Login
	PasswordChanged
	EmailChanged
	RoleChanged
)

var ActionState = map[Action]string{
	Unspecified:     "unspecified",
	Created:         "created",
	Updated:         "updated",
	Deleted:         "deleted",
	Restored:        "restored",
	Login:           "login",
	PasswordChanged: "password_changed",
	EmailChanged:    "email_changed",
	RoleChanged:     "role_changed",
}

// String returns string value of the Action value
func (sa Action) String() string {
	return ActionState[sa]
}

// Valid returns true if value in map else false
func (sa Action) Valid() bool {
	_, ok := ActionState[sa]
	return ok
}

func (sa Action) LastLoginChanges(oldValue, newValue time.Time) map[string]interface{} {
	if sa == Login {
		return map[string]interface{}{
			"last_login": map[string]interface{}{
				"oldValue": oldValue,
				"newValue": newValue,
			},
		}
	}
	return nil
}

func (sa Action) AccountCreatedChanges(createdAt time.Time) map[string]interface{} {
	if sa == Created {
		return map[string]interface{}{
			"created_at": map[string]interface{}{
				"oldValue": nil,
				"newValue": createdAt,
			},
		}
	}
	return nil
}

func (sa Action) PasswordChangedChanges(oldValue, newValue string) map[string]interface{} {
	if sa == PasswordChanged {
		return map[string]interface{}{
			"password_hash": map[string]interface{}{
				"oldValue": oldValue,
				"newValue": newValue,
			},
		}
	}
	return nil
}

func (sa Action) EmailVerifiedChanges() map[string]interface{} {
	if sa == EmailChanged {
		return map[string]interface{}{
			"is_active": map[string]interface{}{
				"oldValue": false,
				"newValue": true,
			},
		}
	}
	return nil
}

func (sa Action) EmailChangedChanges(oldValue, newValue time.Time) map[string]interface{} {
	if sa == EmailChanged {
		return map[string]interface{}{
			"email_reset_at": map[string]interface{}{
				"oldValue": oldValue,
				"newValue": newValue,
			},
		}
	}
	return nil
}

var actionByName = func() map[string]Action {
	m := make(map[string]Action, len(ActionState))
	for k, v := range ActionState {
		m[v] = k
	}
	return m
}()

// ParseAction parse string from database in Action type
// DATABASE FUNCTION
func ParseAction(s string) (Action, bool) {
	a, ok := actionByName[s]
	return a, ok
}

type UserHistory struct {
	ID            int64                  `json:"id"`
	Username      string                 `json:"username"`
	ActorUsername string                 `json:"actor_username"`
	Action        Action                 `json:"action"`
	Changes       map[string]interface{} `json:"changes"`
	IpAddress     *string                `json:"ip_address"`
	UserAgent     *string                `json:"user_agent"`
	RequestId     *string                `json:"request_id"`
	CreatedAt     time.Time              `json:"created_at"`
}

type CreateUserHistoryParams struct {
	Username      string                 `json:"username"`
	ActorUsername string                 `json:"actor_username"`
	Action        Action                 `json:"action"`
	Changes       map[string]interface{} `json:"changes"`
	IpAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	RequestId     string                 `json:"request_id"`
}

type ListUserHistoryParams struct {
	Username string `json:"username"`
	Action   Action `json:"action"`
	BeforeId int64  `json:"before_id"` // id of the last record previous page
	PageSize int32  `json:"page_size"`
}

type ListUserHistoryByActorParams struct {
	ActorUsername string `json:"actor_username"`
	BeforeId      int64  `json:"before_id"`
	PageSize      int32  `json:"page_size"`
}

type UserHistoryRepository interface {
	CreateUserHistory(ctx context.Context, p *CreateUserHistoryParams) (*UserHistory, error)
	GetUserHistory(ctx context.Context, id int64) (*UserHistory, error)
	ListUserHistory(ctx context.Context, p *ListUserHistoryParams) ([]UserHistory, error)
	ListUserHistoryByActor(ctx context.Context, p *ListUserHistoryByActorParams) ([]UserHistory, error)
	CountUserHistory(ctx context.Context, username string) (int64, error)
	DeleteUserHistoryOlderThan(ctx context.Context, targetDate time.Time) (int64, error)
}
