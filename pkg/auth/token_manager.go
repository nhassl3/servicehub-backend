package auth

type CreateTokenParams struct {
	Username string `json:"username"`
	UID      string `json:"uid"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	IsActive bool   `json:"isActive"`
}

// TokenManager defines the interface for token creation and verification.
type TokenManager interface {
	CreateToken(params *CreateTokenParams) (string, error)
	CreateRefreshToken(params *CreateTokenParams) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}
