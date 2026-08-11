package grpc

import (
	"context"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	adminv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/admin/v1"
	productv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/product/v1"
)

// AdminHandler implements adminv1.AdminServiceServer.
//
// Implemented RPC methods:
//   - CreateAdmin
//   - GetAdminProfile
//   - UpdateAdminProfile
//   - UploadAvatar
//   - IncreaseTotalModerates
//   - GetAdminStatistics
type AdminHandler struct {
	adminv1.UnimplementedAdminServiceServer
	svc       *service.AdminService
	analytics *service.AnalyticsService
}

func NewAdminHandler(svc *service.AdminService, analytics *service.AnalyticsService) *AdminHandler {
	return &AdminHandler{svc: svc, analytics: analytics}
}

func (h *AdminHandler) CreateAdmin(ctx context.Context, req *adminv1.CreateAdminRequest) (*adminv1.CreateAdminResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	admin, err := h.svc.CreateAdmin(ctx, domain.CreateAdminParams{
		Username:    username,
		DisplayName: req.GetDisplayName(),
		LevelRights: req.GetLevelRights(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &adminv1.CreateAdminResponse{Admin: protoAdmin(admin)}, nil
}

func (h *AdminHandler) GetAdminProfile(ctx context.Context, req *adminv1.GetAdminProfileRequest) (*adminv1.GetAdminProfileResponse, error) {
	admin, err := h.svc.GetAdminProfile(ctx, domain.GetAdminProfileParams{
		Username: req.GetUsername(),
	})
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

func (h *AdminHandler) GetAdminStatistics(ctx context.Context, req *adminv1.GetAdminStatisticsRequest) (*adminv1.GetAdminStatisticsResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}

	params := domain.AdminStatisticsParams{Granularity: req.GetGranularity()}
	if req.GetFrom() != nil {
		params.From = req.GetFrom().AsTime()
	}
	if req.GetTo() != nil {
		params.To = req.GetTo().AsTime()
	}
	now := time.Now()
	if params.From.IsZero() {
		params.From = now.AddDate(0, 0, -30) // default: last 30 days
	}
	if params.To.IsZero() {
		params.To = now
	}

	stats, err := h.analytics.GetStatistics(ctx, username, params)
	if err != nil {
		return nil, domainErr(err)
	}
	return &adminv1.GetAdminStatisticsResponse{
		Products: &adminv1.ProductStatusStats{
			VerifiedCount: int32(stats.Products.Verified),
			PendingCount:  int32(stats.Products.Pending),
			RejectedCount: int32(stats.Products.Rejected),
		},
		TopProducts:   protoTopProducts(stats.TopProducts),
		TopCategories: protoTopCategories(stats.TopCategories),
		Registrations: protoRegistrations(stats.Registrations),
		Moderates:     protoModerates(stats.Moderates),
	}, nil
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

func protoTopProducts(top []domain.TopProduct) []*adminv1.TopProduct {
	res := make([]*adminv1.TopProduct, 0, len(top))
	for _, p := range top {
		res = append(res, &adminv1.TopProduct{
			Product: &productv1.Product{
				Id:           p.ID,
				CategoryId:   int32(p.CategoryID),
				Title:        p.Title,
				Rating:       p.Rating,
				SalesCount:   int32(p.SalesCount),
				ReviewsCount: int32(p.ReviewsCount),
			},
		})
	}
	return res
}

func protoTopCategories(cats []domain.CategorySales) []*adminv1.CategorySales {
	res := make([]*adminv1.CategorySales, 0, len(cats))
	for _, c := range cats {
		res = append(res, &adminv1.CategorySales{
			CategoryId: int32(c.CategoryID),
			Name:       c.CategoryName,
			SalesCount: int32(c.SalesCount),
		})
	}
	return res
}

func protoRegistrations(pts []domain.RegistrationPoint) []*adminv1.RegistrationPoint {
	res := make([]*adminv1.RegistrationPoint, 0, len(pts))
	for _, p := range pts {
		res = append(res, &adminv1.RegistrationPoint{
			Bucket: safeTimestamp(p.Bucket),
			Count:  int32(p.Count),
		})
	}
	return res
}

func protoModerates(pts []domain.ModeratePoint) []*adminv1.ModeratePoint {
	res := make([]*adminv1.ModeratePoint, 0, len(pts))
	for _, p := range pts {
		res = append(res, &adminv1.ModeratePoint{
			Bucket:    safeTimestamp(p.Bucket),
			Count:     int32(p.Count),
			AdminId:   p.AdminID,
			AdminName: p.AdminUsername,
		})
	}
	return res
}
