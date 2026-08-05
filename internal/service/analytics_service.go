package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// minStatsLevelRights is the minimum admins.level_rights allowed to read
// aggregate analytics. Admittedly coarse, but the plan explicitly scopes the
// statistics endpoint to admins with level_rights >= 3.
const minStatsLevelRights int32 = 3

// AnalyticsService returns OLAP aggregates for the AdminPanel. Access is
// restricted to administrators with level_rights >= 3.
type AnalyticsService struct {
	analyticsRepo domain.AnalyticsRepository
	adminRepo     domain.AdminRepository
	adminRedis    domain.AdminRedis
}

func NewAnalyticsService(analyticsRepo domain.AnalyticsRepository, adminRepo domain.AdminRepository, adminRedis domain.AdminRedis) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		adminRepo:     adminRepo,
		adminRedis:    adminRedis,
	}
}

// GetStatistics resolves the calling admin, verifies level_rights >= 3 and
// returns the ClickHouse-aggregated statistics for the requested period.
func (s *AnalyticsService) GetStatistics(ctx context.Context, username string, params domain.AdminStatisticsParams) (*domain.AdminStatistics, error) {
	admin, err := s.resolveAdmin(ctx, username)
	if err != nil {
		return nil, err
	}
	if admin.LevelRights < minStatsLevelRights {
		return nil, domain.ErrForbidden
	}

	stats, err := s.analyticsRepo.GetAdminStatistics(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("analytics_service.GetStatistics: %w", err)
	}
	return stats, nil
}

// resolveAdmin mirrors the cache-aside pattern used across the backend:
// read the profile from Redis, fall back to Postgres, and warm the cache.
func (s *AnalyticsService) resolveAdmin(ctx context.Context, username string) (*domain.Admin, error) {
	admin, err := s.adminRedis.Profile(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			admin, err = s.adminRepo.GetAdmin(ctx, domain.GetAdminProfileParams{Username: username})
			if err != nil {
				return nil, fmt.Errorf("analytics_service.resolveAdmin: %w", err)
			}
			_ = s.adminRedis.SetProfile(ctx, admin)
			return admin, nil
		}
		return nil, fmt.Errorf("analytics_service.resolveAdmin (redis): %w", err)
	}
	return admin, nil
}
