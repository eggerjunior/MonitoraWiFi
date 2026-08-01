package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("uma-senha-forte-123")
	if err != nil {
		t.Fatalf("erro ao gerar hash: %v", err)
	}
	if hash == "uma-senha-forte-123" {
		t.Fatalf("hash não pode ser igual à senha em texto claro")
	}
	if !VerifyPassword(hash, "uma-senha-forte-123") {
		t.Fatalf("senha correta deveria validar")
	}
	if VerifyPassword(hash, "senha-errada") {
		t.Fatalf("senha errada não deveria validar")
	}
}

func TestGenerateSessionToken_HashIsDeterministicAndTokenIsNotStoredDirectly(t *testing.T) {
	token, tokenHash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("erro ao gerar token: %v", err)
	}
	if token == "" || tokenHash == "" {
		t.Fatalf("token e hash não podem ser vazios")
	}
	if token == tokenHash {
		t.Fatalf("o valor persistido deve ser o hash, não o token em claro")
	}
	if HashSessionToken(token) != tokenHash {
		t.Fatalf("hash do token deve ser determinístico para permitir lookup por hash")
	}
}

func TestGenerateSessionToken_IsUnpredictable(t *testing.T) {
	t1, _, _ := GenerateSessionToken()
	t2, _, _ := GenerateSessionToken()
	if t1 == t2 {
		t.Fatalf("dois tokens gerados não podem ser iguais")
	}
}
