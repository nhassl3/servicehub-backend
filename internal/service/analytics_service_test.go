package service_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	mockrepo "github.com/nhassl3/servicehub-backend/internal/repository/mock"
	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/stretchr/testify/require"
)

const statsMinLevel = 3

func newAnalyticsSvc(t *testing.T) (*service.AnalyticsService, *mockrepo.MockAdminRedis, *mockrepo.MockAdminRepository, *mockrepo.MockAnalyticsRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	adminRedis := mockrepo.NewMockAdminRedis(ctrl)
	adminRepo := mockrepo.NewMockAdminRepository(ctrl)
	analyticsRepo := mockrepo.NewMockAnalyticsRepository(ctrl)
	svc := service.NewAnalyticsService(analyticsRepo, adminRepo, adminRedis)
	return svc, adminRedis, adminRepo, analyticsRepo
}

func TestAnalyticsService_GetStatistics_AdminLevelOK(t *testing.T) {
	svc, adminRedis, _, analyticsRepo := newAnalyticsSvc(t)
	ctx := context.Background()

	adminRedis.EXPECT().Profile(ctx, "boss").Return(&domain.Admin{Username: "boss", LevelRights: statsMinLevel}, nil)
	analyticsRepo.EXPECT().GetAdminStatistics(ctx, gomock.Any()).Return(&domain.AdminStatistics{
		Products: domain.ProductStatusStats{Verified: 10, Pending: 2, Rejected: 1},
	}, nil)

	res, err := svc.GetStatistics(ctx, "boss", domain.AdminStatisticsParams{})
	require.NoError(t, err)
	require.Equal(t, 10, res.Products.Verified)
}

func TestAnalyticsService_GetStatistics_LevelBelowThreshold(t *testing.T) {
	svc, adminRedis, _, analyticsRepo := newAnalyticsSvc(t)
	ctx := context.Background()

	adminRedis.EXPECT().Profile(ctx, "low").Return(&domain.Admin{Username: "low", LevelRights: 2}, nil)
	analyticsRepo.EXPECT().GetAdminStatistics(ctx, gomock.Any()).Times(0)

	_, err := svc.GetStatistics(ctx, "low", domain.AdminStatisticsParams{})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAnalyticsService_GetStatistics_NotAdmin(t *testing.T) {
	svc, adminRedis, adminRepo, analyticsRepo := newAnalyticsSvc(t)
	ctx := context.Background()

	// Cache miss falls back to Postgres, which says the user is not an admin.
	adminRedis.EXPECT().Profile(ctx, "nobody").Return(nil, domain.ErrRedisNotFound)
	adminRepo.EXPECT().GetAdmin(ctx, gomock.Any()).Return(nil, domain.ErrNotFound)
	analyticsRepo.EXPECT().GetAdminStatistics(ctx, gomock.Any()).Times(0)

	_, err := svc.GetStatistics(ctx, "nobody", domain.AdminStatisticsParams{})
	require.Error(t, err)
}

func TestAnalyticsService_GetStatistics_ReportsClickHouseError(t *testing.T) {
	svc, adminRedis, _, analyticsRepo := newAnalyticsSvc(t)
	ctx := context.Background()

	adminRedis.EXPECT().Profile(ctx, "analyst").Return(&domain.Admin{Username: "analyst", LevelRights: statsMinLevel}, nil)
	analyticsRepo.EXPECT().GetAdminStatistics(ctx, gomock.Any()).Return(nil, context.DeadlineExceeded)

	_, err := svc.GetStatistics(ctx, "analyst", domain.AdminStatisticsParams{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
