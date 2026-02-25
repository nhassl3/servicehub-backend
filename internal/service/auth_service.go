package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/pkg/auth"
	"github.com/nhassl3/servicehub/pkg/hash"
)

type AuthService struct {
	userRepo       domain.UserRepository
	tokenManager   auth.TokenManager
	refreshManager auth.TokenManager
}

func NewAuthService(userRepo domain.UserRepository, tokenManager, refreshManager auth.TokenManager) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenManager:   tokenManager,
		refreshManager: refreshManager,
	}
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
	FullName string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*domain.User, *TokenPair, error) {
	existsUsername, err := s.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register check username: %w", err)
	}
	if existsUsername {
		return nil, nil, domain.ErrAlreadyExists
	}

	existsEmail, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register check email: %w", err)
	}
	if existsEmail {
		return nil, nil, domain.ErrAlreadyExists
	}

	passwordHash, err := hash.CreateHashPassword(input.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register hash: %w", err)
	}

	user, err := s.userRepo.Create(ctx, domain.CreateUserParams{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     input.FullName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register create: %w", err)
	}

	tokens, err := s.createTokenPair(user.Username, user.UID, user.Role)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, *TokenPair, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("auth_service.Login get user: %w", err)
	}

	ok, err := hash.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Login verify: %w", err)
	}
	if !ok {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.createTokenPair(user.Username, user.UID, user.Role)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	payload, err := s.refreshManager.VerifyToken(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return s.createTokenPair(payload.Username, payload.UID, payload.Role)
}

func (s *AuthService) GetMe(ctx context.Context, username string) (*domain.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}

func (s *AuthService) createTokenPair(username, uid, role string) (*TokenPair, error) {
	accessToken, err := s.tokenManager.CreateToken(username, uid, role)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create access token: %w", err)
	}

	refreshToken, err := s.refreshManager.CreateToken(username, uid, role)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
