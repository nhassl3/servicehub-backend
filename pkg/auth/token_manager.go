package auth

// TokenManager defines the interface for token creation and verification.
type TokenManager interface {
	CreateToken(username, uid, role string) (string, error)
	VerifyToken(token string) (*Payload, error)
}
