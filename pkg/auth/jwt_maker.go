package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTMaker implements TokenManager using HMAC-SHA256 JWT tokens.
type JWTMaker struct {
	secret []byte
	ttl    time.Duration
}

type jwtClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	Email    string `json:"email"`
	UID      string `json:"uid"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

// NewJWTMaker creates a new JWTMaker.
func NewJWTMaker(secret string, ttl time.Duration) (*JWTMaker, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt: secret must be at least 32 characters")
	}
	return &JWTMaker{secret: []byte(secret), ttl: ttl}, nil
}

func (m *JWTMaker) CreateToken(username, uid, role, email string, isActive bool) (string, error) {
	return m.createToken(username, uid, role, email, uuid.New().String(), isActive, time.Now())
}

func (m *JWTMaker) CreateRefreshToken(username, uid, role, email string, isActive bool) (string, *Payload, error) {
	begin := time.Now()
	jti := uuid.New().String()
	token, err := m.createToken(username, uid, role, email, jti, isActive, begin)
	if err != nil {
		return "", nil, err
	}
	return token, &Payload{
		JTI:       jti,
		Username:  username,
		Email:     email,
		UID:       uid,
		Role:      role,
		IsActive:  isActive,
		IssuedAt:  begin,
		ExpiredAt: begin.Add(m.ttl),
	}, nil
}

func (m *JWTMaker) createToken(username, uid, role, email, jti string, isActive bool, start time.Time) (string, error) {
	if start.Equal(time.Time{}) {
		start = time.Now()
	}

	claims := &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(start),
			ExpiresAt: jwt.NewNumericDate(start.Add(m.ttl)),
		},
		Username: username,
		Email:    email,
		UID:      uid,
		Role:     role,
		IsActive: isActive,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (m *JWTMaker) VerifyToken(tokenStr string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}

	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return &Payload{
		JTI:       claims.ID,
		Username:  claims.Username,
		Email:     claims.Email,
		UID:       claims.UID,
		Role:      claims.Role,
		IsActive:  claims.IsActive,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiredAt: claims.ExpiresAt.Time,
	}, nil
}
