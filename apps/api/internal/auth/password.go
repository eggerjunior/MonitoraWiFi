// Package auth cobre hashing de senha, geração/validação de token de sessão e a
// matriz de permissões RBAC (Seção 18).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword usa bcrypt (custo padrão da lib, 10) — adequado para senha de
// usuário; nunca armazenar senha em texto claro (Seção 2.2).
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("gerar hash de senha: %w", err)
	}
	return string(hash), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// GenerateSessionToken cria um token opaco aleatório (não um JWT) para a sessão
// web/iOS. O valor devolvido ao cliente é o token; o backend só persiste o hash
// SHA-256 dele, para que um vazamento do banco não exponha tokens válidos.
func GenerateSessionToken() (token string, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("gerar token de sessão: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	tokenHash = HashSessionToken(token)
	return token, tokenHash, nil
}

func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
