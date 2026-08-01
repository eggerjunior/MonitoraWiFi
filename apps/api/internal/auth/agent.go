package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateAgentSecret e GenerateEnrollmentToken usam o mesmo esquema de
// GenerateSessionToken (token opaco aleatório, hash SHA-256 persistido) —
// nunca guardamos o segredo em texto claro, só o hash (Seção 2.2).

func GenerateAgentSecret() (secret string, secretHash string, err error) {
	return generateOpaqueToken()
}

func HashAgentSecret(secret string) string {
	return hashOpaqueToken(secret)
}

func GenerateEnrollmentToken() (token string, tokenHash string, err error) {
	return generateOpaqueToken()
}

func HashEnrollmentToken(token string) string {
	return hashOpaqueToken(token)
}

func generateOpaqueToken() (token string, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("gerar token opaco: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	tokenHash = hashOpaqueToken(token)
	return token, tokenHash, nil
}

func hashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
