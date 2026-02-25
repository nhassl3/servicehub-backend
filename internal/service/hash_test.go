package service_test

import (
	"testing"

	"github.com/nhassl3/servicehub/pkg/hash"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_VerifyPassword(t *testing.T) {
	password := "mySecurePassword!123"

	h, err := hash.HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, h)
	require.NotEqual(t, password, h)

	ok, err := hash.VerifyPassword(password, h)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	h, _ := hash.HashPassword("correct")
	ok, err := hash.VerifyPassword("wrong", h)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	_, err := hash.VerifyPassword("any", "not-a-valid-hash")
	require.ErrorIs(t, err, hash.ErrInvalidHash)
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	h1, _ := hash.HashPassword("same")
	h2, _ := hash.HashPassword("same")
	require.NotEqual(t, h1, h2, "same password should produce different hashes due to random salt")
}
