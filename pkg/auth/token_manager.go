package auth

// TokenManager defines the interface for token creation and verification.
type TokenManager interface {
	CreateToken(username, uid, role, email string, isActive bool) (string, error)
	CreateRefreshToken(username, uid, role, email string, isActive bool) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}
