package auth

import "errors"

var (
	ErrExpiredToken = errors.New("token has expired")
	ErrInvalidToken = errors.New("token is invalid")
	ErrTokenRevoked = errors.New("token has been revoked")
)

// IsAny checks if a catch error in slices of errors with typically errors for Token
// Some errors from this slice: ErrExpiredToken, ErrInvalidToken, ErrTokenRevoked
func IsAny(err error) bool {
	for _, t := range []error{ErrTokenRevoked, ErrInvalidToken, ErrExpiredToken} {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}
