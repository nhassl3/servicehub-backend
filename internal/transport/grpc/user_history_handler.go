package grpc

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	pb "github.com/nhassl3/servicehub-contracts/pkg/pb/user_history/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHistoryHandler struct {
	pb.UnimplementedUserHistoryServiceServer
	svc *service.UserHistoryService
}

func NewUserHistoryHandler(svc *service.UserHistoryService) *UserHistoryHandler {
	return &UserHistoryHandler{
		svc: svc,
	}
}

func (proto *UserHistoryHandler) ListUserHistory(ctx context.Context, req *pb.ListUserHistoryRequest) (*pb.ListUserHistoryResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	rows, err := proto.svc.ListUserHistory(ctx, req, service.NewCaller(payload))
	if err != nil {
		return nil, domainErr(err)
	}
	return rows, nil
}

func (proto *UserHistoryHandler) GetUserHistoryEntry(ctx context.Context, req *pb.GetUserHistoryEntryRequest) (*pb.GetUserHistoryEntryResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	row, err := proto.svc.GetUserHistory(ctx, req, service.NewCaller(payload))
	if err != nil {
		return nil, domainErr(err)
	}
	return row, nil
}

func (proto *UserHistoryHandler) ListActorActivity(ctx context.Context, req *pb.ListActorActivityRequest) (*pb.ListActorActivityResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	rows, err := proto.svc.ListActorActivity(ctx, req, service.NewCaller(payload))
	if err != nil {
		return nil, domainErr(err)
	}
	return rows, nil
}
