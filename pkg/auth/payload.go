package auth

import (
	"time"
)

// Payload holds the claims embedded in a token.
type Payload struct {
	Username  string    `json:"username"`
	UID       string    `json:"uid"`
	Role      string    `json:"role"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// Valid checks whether the token payload has not expired.
func (p *Payload) Valid() error {
	if time.Now().After(p.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
