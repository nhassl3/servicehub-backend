package service_test

import (
	"testing"
	"time"

	"github.com/nhassl3/servicehub/pkg/auth"
	"github.com/stretchr/testify/require"
)

const testPasetoKey = "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"

func TestPasetoMaker_CreateAndVerify(t *testing.T) {
	maker, err := auth.NewPasetoMaker(testPasetoKey, 15*time.Minute)
	require.NoError(t, err)

	token, err := maker.CreateToken("alice", "uid-123", "buyer")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.Equal(t, "alice", payload.Username)
	require.Equal(t, "uid-123", payload.UID)
	require.Equal(t, "buyer", payload.Role)
	require.WithinDuration(t, time.Now().Add(15*time.Minute), payload.ExpiredAt, 5*time.Second)
}

func TestPasetoMaker_ExpiredToken(t *testing.T) {
	maker, _ := auth.NewPasetoMaker(testPasetoKey, -1*time.Second)
	token, _ := maker.CreateToken("alice", "uid-123", "buyer")

	_, err := maker.VerifyToken(token)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestPasetoMaker_InvalidToken(t *testing.T) {
	maker, _ := auth.NewPasetoMaker(testPasetoKey, 15*time.Minute)
	_, err := maker.VerifyToken("not-a-valid-paseto-token")
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestPasetoMaker_InvalidKeyLength(t *testing.T) {
	_, err := auth.NewPasetoMaker("tooshort", 15*time.Minute)
	require.Error(t, err)
}

func TestJWTMaker_CreateAndVerify(t *testing.T) {
	maker, err := auth.NewJWTMaker("supersecretkeythatisatleast32chars!!", 15*time.Minute)
	require.NoError(t, err)

	token, err := maker.CreateToken("bob", "uid-456", "seller")
	require.NoError(t, err)

	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.Equal(t, "bob", payload.Username)
	require.Equal(t, "seller", payload.Role)
}

func TestJWTMaker_ExpiredToken(t *testing.T) {
	maker, _ := auth.NewJWTMaker("supersecretkeythatisatleast32chars!!", -1*time.Second)
	token, _ := maker.CreateToken("bob", "uid", "buyer")
	_, err := maker.VerifyToken(token)
	require.ErrorIs(t, err, auth.ErrExpiredToken)
}
