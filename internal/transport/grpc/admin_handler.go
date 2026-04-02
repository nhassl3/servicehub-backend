package grpc

import (
	"context"

	adminv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/admin/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
)

// AdminHandler implements adminv1.AdminServiceServer.
//
// Implemented RPC methods:
//   - CreateAdmin
//   - GetAdminProfile
//   - UpdateAdminProfile
//   - UploadAvatar
//   - IncreaseTotalModerates
type AdminHandler struct {
	adminv1.UnimplementedAdminServiceServer
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) CreateAdmin(ctx context.Context, req *adminv1.CreateAdminRequest) (*adminv1.CreateAdminResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	admin, err := h.svc.CreateAdmin(ctx, domain.CreateAdminParams{
		Username:    username,
		DisplayName: req.GetDisplayName(),
		LevelRights: int32(req.GetLevelRights()),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &adminv1.CreateAdminResponse{Admin: protoAdmin(admin)}, nil
}

func (h *AdminHandler) GetAdminProfile(ctx context.Context, req *adminv1.GetAdminProfileRequest) (*adminv1.GetAdminProfileResponse, error) {
	var params domain.GetAdminProfileParams

	if req.Username != nil {
		v := *req.Username
		params.Username = &v
	}
	if req.AdminId != nil {
		v := *req.AdminId
		params.AdminId = &v
	}

	admin, err := h.svc.GetAdminProfile(ctx, params)
	if err != nil {
		return nil, domainErr(err)
	}
	return &adminv1.GetAdminProfileResponse{Admin: protoAdmin(admin)}, nil
}

func (h *AdminHandler) UpdateAdminProfile(ctx context.Context, req *adminv1.UpdateAdminProfileRequest) (*adminv1.UpdateAdminProfileResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, domainErr(err)
	}
	admin, err := h.svc.UpdateAdminProfile(ctx, domain.UpdateAdminsProfileParams{
		Username:       username,
		DisplayName:    req.DisplayName,
		LevelRights:    req.LevelRights,
		TotalModerates: req.TotalModeration,
		AvatarURL:      req.AvatarUrl,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &adminv1.UpdateAdminProfileResponse{Admin: protoAdmin(admin)}, nil
}

func (h *AdminHandler) UploadAvatar(ctx context.Context, req *adminv1.UploadAvatarRequest) (*adminv1.UploadAvatarResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, domainErr(err)
	}
	admin, err := h.svc.UploadAdminAvatar(ctx, domain.UploadAdminAvatar{
		Username:    username,
		FileData:    req.FileData,
		ContentType: req.ContentType,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &adminv1.UploadAvatarResponse{
		Admin: protoAdmin(admin),
	}, nil
}

func (h *AdminHandler) IncreaseTotalModerates(ctx context.Context, req *adminv1.IncreaseAdminModeratesRequest) (*adminv1.IncreaseAdminModeratesResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, domainErr(err)
	}
	return nil, h.svc.IncreaseTotalModerates(ctx, domain.IncreaseTotalModeratesParams{
		Username:       username,
		TotalModerates: req.TotalModeration,
	})
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoAdmin(s *domain.Admin) *adminv1.AdminProfile {
	return &adminv1.AdminProfile{
		Id:              s.ID,
		Username:        s.Username,
		DisplayName:     s.DisplayName,
		LevelRights:     s.LevelRights,
		TotalModeration: s.TotalModerates,
		AvatarUrl:       s.AvatarURL,
		CreatedAt:       safeTimestamp(s.CreatedAt),
		UpdatedAt:       safeTimestamp(s.UpdatedAt),
	}
}
