package auth

import (
	"encoding/hex"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

// PasetoMaker implements TokenManager using PASETO v4 local tokens.
type PasetoMaker struct {
	key paseto.V4SymmetricKey
	ttl time.Duration
}

// NewPasetoMaker creates a new PasetoMaker. keyHex must be a 32-byte hex-encoded string.
func NewPasetoMaker(keyHex string, ttl time.Duration) (*PasetoMaker, error) {
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("paseto: decode key: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("paseto: key must be exactly 32 bytes, got %d", len(keyBytes))
	}

	key, err := paseto.V4SymmetricKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("paseto: create key: %w", err)
	}

	return &PasetoMaker{key: key, ttl: ttl}, nil
}

func (m *PasetoMaker) CreateToken(username, uid, role string) (string, error) {
	return m.createToken(username, uid, role, time.Now())
}

func (m *PasetoMaker) CreateRefreshToken(username, uid, role string) (string, *Payload, error) {
	begin := time.Now()
	token, err := m.createToken(username, uid, role, begin)
	if err != nil {
		return "", nil, err
	}
	return token, &Payload{
		Username:  username,
		UID:       uid,
		Role:      role,
		IssuedAt:  begin,
		ExpiredAt: begin.Add(m.ttl),
	}, nil
}

func (m *PasetoMaker) createToken(username, uid, role string, startTime time.Time) (string, error) {
	if startTime == (time.Time{}) {
		startTime = time.Now()
	}

	token := paseto.NewToken()
	token.SetJti(uuid.New().String())
	token.SetIssuedAt(startTime)
	token.SetExpiration(startTime.Add(m.ttl))
	token.SetString("username", username)
	token.SetString("uid", uid)
	token.SetString("role", role)

	encrypted := token.V4Encrypt(m.key, nil)
	return encrypted, nil
}

func (m *PasetoMaker) VerifyToken(tokenStr string) (*Payload, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())

	token, err := parser.ParseV4Local(m.key, tokenStr, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	username, err := token.GetString("username")
	if err != nil {
		return nil, ErrInvalidToken
	}

	uid, err := token.GetString("uid")
	if err != nil {
		return nil, ErrInvalidToken
	}

	role, err := token.GetString("role")
	if err != nil {
		return nil, ErrInvalidToken
	}

	issuedAt, err := token.GetIssuedAt()
	if err != nil {
		return nil, ErrInvalidToken
	}

	expiredAt, err := token.GetExpiration()
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &Payload{
		Username:  username,
		UID:       uid,
		Role:      role,
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}, nil
}
