package service

import (
	"github.com/nhassl3/servicehub/internal/domain"
)

type NotificationService struct {
	repo *domain.NotificationRepository
}

func NewNotificationService(repo *domain.NotificationRepository) *NotificationService {
	return &NotificationService{
		repo: repo,
	}
}
