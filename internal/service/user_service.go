package service

import (
	"context"
	"fmt"

	"github.com/go-passwd/validator"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/pkg/hash"
)

type UserService struct {
	userRepo domain.UserRepository
}

func NewUserService(userRepo domain.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetUser(ctx context.Context, username string) (*domain.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}

func (s *UserService) UpdateProfile(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error) {
	return s.userRepo.Update(ctx, params)
}

func (s *UserService) UpdatePassword(ctx context.Context, params domain.UpdateUserPasswordParams) (*domain.User, error) {
	user, err := s.GetUser(ctx, params.Username)
	if err != nil {
		return nil, err
	}

	ok, err := hash.VerifyPassword(params.OldPassword, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("user_service.UpdatePassword verify password: %w", err)
	}
	if !ok {
		return nil, domain.ErrPasswordDontMatch
	}

	// similarity by default is 0.7 - DO NOT CHANGE THIS THRESHOLD
	passwordValidator := validator.New(
		validator.Similarity([]string{params.OldPassword}, nil, domain.ErrTooSimilarPasswords),
	)
	if err := passwordValidator.Validate(params.NewPassword); err != nil {
		return nil, err
	}

	newPasswordHashed, err := hash.CreateHashPassword(params.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("user_service.UpdatePassword.NewPassword: %w", err)
	}

	params = domain.UpdateUserPasswordParams{
		Username:    params.Username,
		NewPassword: newPasswordHashed,
	}

	return s.userRepo.UpdatePassword(ctx, params)
}
