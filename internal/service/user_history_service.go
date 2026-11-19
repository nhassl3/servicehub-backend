package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/nhassl3/servicehub-backend/pkg/errorcreator"
	pb "github.com/nhassl3/servicehub-contracts/pkg/pb/user_history/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
	op              = "user_history_service"
)

type UserHistoryService struct {
	userHistoryRepo domain.UserHistoryRepository
}

func NewUserHistoryService(repo domain.UserHistoryRepository) *UserHistoryService {
	return &UserHistoryService{
		userHistoryRepo: repo,
	}
}

type Caller struct {
	Username string
	IsAdmin  bool
}

// NewCaller returns object info about caller
func NewCaller(payload *auth.Payload) *Caller {
	if payload.Role == "admin" {
		return &Caller{
			Username: payload.Username,
			IsAdmin:  true,
		}
	}
	return &Caller{
		Username: payload.Username,
	}
}

func (s *UserHistoryService) ListUserHistory(ctx context.Context, req *pb.ListUserHistoryRequest, caller *Caller) (*pb.ListUserHistoryResponse, error) {
	if caller.Username != req.GetUsername() && !caller.IsAdmin {
		return nil, domain.ErrForbidden
	}
	beforeId, err := decodeToken(req.GetPageToken())
	if err != nil {
		return nil, errorcreator.NewError(op, "ListUserHistory", "failed to decode token", err)
	}
	size := normalizePageSize(req.GetPageSize())
	listUserHistory, err := s.userHistoryRepo.ListUserHistory(ctx, new(domain.ListUserHistoryParams{
		Username: req.GetUsername(),
		Action:   ActionFromProto(req.GetAction()),
		BeforeId: beforeId,
		PageSize: size + 1,
	}))
	if err != nil {
		return nil, errorcreator.NewError(op, "ListUserHistory", "failed to list user history", err)
	}
	rows, next := paginate(listUserHistory, size, func(r domain.UserHistory) int64 { return r.ID })
	listUserHistoryPb := protoUserHistories(rows, caller)
	return new(pb.ListUserHistoryResponse{
		Entries:       listUserHistoryPb,
		NextPageToken: next,
	}), nil
}

func (s *UserHistoryService) ListUserHistoryByActor(ctx context.Context, req *domain.ListUserHistoryByActorParams, caller *Caller) ([]domain.UserHistory, error) {
	if caller.Username != req.ActorUsername && !caller.IsAdmin {
		return nil, domain.ErrForbidden
	}
	size := normalizePageSize(req.PageSize)
	listUserHistory, err := s.userHistoryRepo.ListUserHistoryByActor(ctx, new(domain.ListUserHistoryByActorParams{
		ActorUsername: req.ActorUsername,
		PageSize:      size + 1,
		BeforeId:      req.BeforeId,
	}))
	if err != nil {
		return nil, errorcreator.NewError(op, "ListUserHistoryByActor", "failed to list user history", err)
	}
	return listUserHistory, nil
}

func (s *UserHistoryService) GetUserHistory(ctx context.Context, req *pb.GetUserHistoryEntryRequest, caller *Caller) (*pb.GetUserHistoryEntryResponse, error) {
	row, err := s.userHistoryRepo.GetUserHistory(ctx, req.GetId())
	if err != nil {
		return nil, errorcreator.NewError(op, "GetUserHistory", "failed to get user history", err)
	}
	if row.Username != caller.Username && !caller.IsAdmin {
		return nil, domain.ErrForbidden
	}
	return &pb.GetUserHistoryEntryResponse{
		Entry: protoUserHistory(row, caller),
	}, nil
}

func (s *UserHistoryService) ListActorActivity(ctx context.Context, req *pb.ListActorActivityRequest, caller *Caller) (*pb.ListActorActivityResponse, error) {
	if !caller.IsAdmin {
		return nil, domain.ErrForbidden
	}

	size := normalizePageSize(req.GetPageSize())
	beforeId, err := decodeToken(req.GetPageToken())
	if err != nil {
		return nil, errorcreator.NewError(op, "ListActorActivity", "failed to decode token", err)
	}
	rows, err := s.userHistoryRepo.ListUserHistoryByActor(ctx, new(domain.ListUserHistoryByActorParams{
		ActorUsername: req.GetActorUsername(),
		BeforeId:      beforeId,
		PageSize:      size + 1,
	}))
	if err != nil {
		return nil, errorcreator.NewError(op, "ListActorActivity", "failed to list user history", err)
	}
	rows, next := paginate(rows, size, func(r domain.UserHistory) int64 { return r.ID })
	return &pb.ListActorActivityResponse{
		Entries:       protoUserHistories(rows, caller),
		NextPageToken: next,
	}, nil
}

func (s *UserHistoryService) CountUserHistory(ctx context.Context, username string, caller *Caller) (int64, error) {
	if caller.Username != username && !caller.IsAdmin {
		return 0, domain.ErrForbidden
	}
	return s.userHistoryRepo.CountUserHistory(ctx, username)
}

func (s *UserHistoryService) DeleteUserHistoryOlderThan(ctx context.Context, beforeTime time.Time, caller *Caller) (int64, error) {
	if !caller.IsAdmin {
		return 0, domain.ErrForbidden
	}
	countDeleted, err := s.userHistoryRepo.DeleteUserHistoryOlderThan(ctx, beforeTime)
	if err != nil {
		return 0, errorcreator.NewError(op, "DeleteUserHistoryOlderThan", "failed to delete user history", err)
	}
	return countDeleted, nil
}

// helpers

func normalizePageSize(n int32) int32 {
	switch {
	case n <= 0:
		return defaultPageSize
	case n > maxPageSize:
		return maxPageSize
	}
	return n
}

// paginate splits not needed (size+1) record and building token next page
func paginate[T any](rows []T, size int32, id func(T) int64) ([]T, string) {
	if int32(len(rows)) <= size {
		return rows, ""
	}
	rows = rows[:size]
	return rows, encodeToken(id(rows[len(rows)-1]))
}

func encodeToken(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func decodeToken(token string) (int64, error) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errorcreator.NewError(op, "decodeToken", "invalid token", domain.ErrInvalidToken)
	}
	return id, nil
}

// Proto mapper

var actionToProtoMap = map[domain.Action]pb.UserHistoryAction{
	domain.Created:         pb.UserHistoryAction_USER_HISTORY_ACTION_CREATED,
	domain.Updated:         pb.UserHistoryAction_USER_HISTORY_ACTION_UPDATED,
	domain.Deleted:         pb.UserHistoryAction_USER_HISTORY_ACTION_DELETED,
	domain.Restored:        pb.UserHistoryAction_USER_HISTORY_ACTION_RESTORED,
	domain.Login:           pb.UserHistoryAction_USER_HISTORY_ACTION_LOGIN,
	domain.PasswordChanged: pb.UserHistoryAction_USER_HISTORY_ACTION_PASSWORD_CHANGED,
	domain.EmailChanged:    pb.UserHistoryAction_USER_HISTORY_ACTION_EMAIL_CHANGED,
	domain.RoleChanged:     pb.UserHistoryAction_USER_HISTORY_ACTION_ROLE_CHANGED,
}

var actionFromProtoMap = func() map[pb.UserHistoryAction]domain.Action {
	m := make(map[pb.UserHistoryAction]domain.Action, len(actionToProtoMap))
	for a, p := range actionToProtoMap {
		m[p] = a
	}
	return m
}()

// ActionToProto неизвестное значение -> UNSPECIFIED.
func ActionToProto(a domain.Action) pb.UserHistoryAction {
	if p, ok := actionToProtoMap[a]; ok {
		return p
	}
	return pb.UserHistoryAction_USER_HISTORY_ACTION_UNSPECIFIED
}

// ActionFromProto UNSPECIFIED и неизвестные значения -> ok=false.
func ActionFromProto(p pb.UserHistoryAction) domain.Action {
	if a, ok := actionFromProtoMap[p]; ok {
		return a
	}
	return domain.Unspecified
}

// fieldChange возврщает историю изменения над конкретным полем
func fieldChange(m map[string]interface{}) ([]*pb.FieldChange, error) {
	res := make([]*pb.FieldChange, 0, len(m))
	for k, v := range m {
		changeMap, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid field change: %v", v)
		}

		oldValue, err := structpb.NewValue(changeMap["oldValue"])
		if err != nil {
			return nil, fmt.Errorf("failed to convert in structpb.Value old value: %v", err)
		}

		newValue, err := structpb.NewValue(changeMap["newValue"])
		if err != nil {
			return nil, fmt.Errorf("failed to convert in structpb.Value new value: %v", err)
		}

		res = append(res, &pb.FieldChange{
			Field:    k,
			OldValue: oldValue,
			NewValue: newValue,
		})
	}
	return res, nil
}

func protoUserHistory(uh *domain.UserHistory, caller *Caller) *pb.UserHistoryEntry {
	fieldChanges, err := fieldChange(uh.Changes)
	if err != nil {
		return new(pb.UserHistoryEntry)
	}

	e := new(pb.UserHistoryEntry{
		Id:            uh.ID,
		Username:      uh.Username,
		ActorUsername: uh.ActorUsername,
		Action:        ActionToProto(uh.Action),
		Changes:       fieldChanges,
		RequestId:     uh.RequestId,
		CreatedAt:     timestamppb.New(uh.CreatedAt),
	})

	if caller.IsAdmin {
		if uh.IpAddress != nil && *uh.IpAddress != "" {
			e.IpAddress = uh.IpAddress
		}
		if uh.UserAgent != nil && *uh.UserAgent != "" {
			e.UserAgent = uh.UserAgent
		}
	}

	return e
}

func protoUserHistories(uhs []domain.UserHistory, caller *Caller) []*pb.UserHistoryEntry {
	res := make([]*pb.UserHistoryEntry, 0, len(uhs))
	for _, uh := range uhs {
		res = append(res, protoUserHistory(&uh, caller))
	}
	return res
}
