package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nhassl3/servicehub-backend/internal/db"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

const op = "user_history_repo"

func newErr(method, msg string, baseError error) error {
	return fmt.Errorf("%s.%s: %s: %w", op, method, msg, baseError)
}

type UserHistoryRepo struct {
	store *db.Store
}

func NewUserHistoryRepo(s *db.Store) *UserHistoryRepo {
	return &UserHistoryRepo{
		store: s,
	}
}

func (repo *UserHistoryRepo) CreateUserHistory(ctx context.Context, p *domain.CreateUserHistoryParams) (*domain.UserHistory, error) {
	if p.Changes == nil {
		return nil, domain.ErrNothingToSave
	}

	bytes, err := json.Marshal(p.Changes)
	if err != nil {
		return nil, newErr("CreateUserHistory", "failed to marshal changes", err)
	}

	userHistory, err := repo.store.CreateUserHistory(ctx, db.CreateUserHistoryParams{
		Username:      p.Username,
		ActorUsername: stringToNullable(p.ActorUsername),
		Action:        actionToDB(p.Action),
		Changes:       bytes,
		IpAddress:     netip.MustParseAddr(p.IpAddress),
		UserAgent:     stringToNullable(p.UserAgent),
		RequestID:     stringToNullable(p.RequestId),
	})
	if err != nil {
		return nil, newErr("CreateUserHistory", "failed to create user history", err)
	}

	if res := mapUserHistory(&userHistory); res != nil {
		return res, nil
	}

	return nil, domain.ErrInvalidInput
}

func (repo *UserHistoryRepo) GetUserHistory(ctx context.Context, id int64) (*domain.UserHistory, error) {
	userHistory, err := repo.store.GetUserHistory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, newErr("GetUserHistory", "failed to get user history", err)
	}
	if res := mapUserHistory(&userHistory); res != nil {
		return res, nil
	}
	return nil, domain.ErrInvalidInput
}

func (repo *UserHistoryRepo) ListUserHistory(ctx context.Context, p *domain.ListUserHistoryParams) ([]domain.UserHistory, error) {
	valid := true
	if p.Action <= 0 {
		valid = false
	}

	var action db.UserHistoryAction
	if valid {
		action = actionToDB(p.Action)
	}

	userHistoryListEntries, err := repo.store.ListUserHistory(ctx, db.ListUserHistoryParams{
		Username: p.Username,
		Action: db.NullUserHistoryAction{
			UserHistoryAction: action,
			Valid:             valid,
		},
		BeforeID: int64PtrToNullable(p.BeforeId),
		PageSize: p.PageSize,
	})
	if err != nil {
		return nil, newErr("ListUserHistory", "failed to list user history", err)
	}

	userHistoryList := mapUserHistories(userHistoryListEntries)

	return userHistoryList, nil
}

func (repo *UserHistoryRepo) ListUserHistoryByActor(ctx context.Context, p *domain.ListUserHistoryByActorParams) ([]domain.UserHistory, error) {
	userHistoryListEntries, err := repo.store.ListUserHistoryByActor(ctx, db.ListUserHistoryByActorParams{
		ActorUsername: stringToNullable(p.ActorUsername),
		BeforeID:      int64PtrToNullable(p.BeforeId),
		PageSize:      p.PageSize,
	})
	if err != nil {
		return nil, newErr("ListUserHistoryByActor", "failed to list user history", err)
	}

	userHistoryList := mapUserHistories(userHistoryListEntries)

	return userHistoryList, nil
}

func (repo *UserHistoryRepo) CountUserHistory(ctx context.Context, username string) (int64, error) {
	countUserHistory, err := repo.store.CountUserHistory(ctx, username)
	if err != nil {
		return 0, newErr("CountUserHistory", "failed to count user history", err)
	}
	return countUserHistory, nil
}

func (repo *UserHistoryRepo) DeleteUserHistoryOlderThan(ctx context.Context, targetDate time.Time) (int64, error) {
	countDeleted, err := repo.store.DeleteUserHistoryOlderThan(ctx, pgtype.Timestamptz{Time: targetDate, Valid: true})
	if err != nil {
		return 0, newErr("DeleteUserHistoryOlderThan", "failed to delete user history", err)
	}
	return countDeleted, nil
}

// domain <-> sqlc-enum (строка в БД).

// actionToDB converts domain.Action to db.UserHistoryAction
func actionToDB(a domain.Action) db.UserHistoryAction {
	return db.UserHistoryAction(a.String())
}

// actionFromDB converts db.UserHistoryAction to domain.Action
func actionFromDB(a db.UserHistoryAction) (domain.Action, bool) {
	return domain.ParseAction(string(a))
}

// mappers

func mapUserHistory(entry *db.UserHistory) *domain.UserHistory {
	action, ok := actionFromDB(entry.Action)
	if !ok {
		return nil
	}

	var changes map[string]interface{}
	if err := json.Unmarshal(entry.Changes, &changes); err != nil {
		return nil
	}

	return new(domain.UserHistory{
		ID:            entry.ID,
		Username:      entry.Username,
		ActorUsername: entry.ActorUsername.String,
		Action:        action,
		Changes:       changes,
		IpAddress:     new(entry.IpAddress.String()),
		UserAgent:     new(entry.UserAgent.String),
		RequestId:     new(entry.RequestID.String),
		CreatedAt:     pgTimeTZ(entry.CreatedAt, time.UTC),
	})
}

func mapUserHistories(entries []db.UserHistory) []domain.UserHistory {
	res := make([]domain.UserHistory, 0, len(entries))
	for _, entry := range entries {
		res = append(res, *mapUserHistory(&entry))
	}
	return res
}
