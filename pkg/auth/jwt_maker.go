package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTMaker implements TokenManager using HMAC-SHA256 JWT tokens.
type JWTMaker struct {
	secret []byte
	ttl    time.Duration
}

type jwtClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	UID      string `json:"uid"`
	Role     string `json:"role"`
}

// NewJWTMaker creates a new JWTMaker.
func NewJWTMaker(secret string, ttl time.Duration) (*JWTMaker, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt: secret must be at least 32 characters")
	}
	return &JWTMaker{secret: []byte(secret), ttl: ttl}, nil
}

func (m *JWTMaker) CreateToken(username, uid, role string) (string, error) {
	now := time.Now()
	claims := &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
		Username: username,
		UID:      uid,
		Role:     role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
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
		Username:  claims.Username,
		UID:       claims.UID,
		Role:      claims.Role,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiredAt: claims.ExpiresAt.Time,
	}, nil
}
