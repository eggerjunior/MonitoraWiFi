package auth

import "testing"

func TestGenerateAgentSecret_UnicoENuncaTextoClaro(t *testing.T) {
	secret1, hash1, err := GenerateAgentSecret()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	secret2, hash2, err := GenerateAgentSecret()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if secret1 == "" || hash1 == "" {
		t.Fatal("esperava secret e hash não vazios")
	}
	if secret1 == secret2 {
		t.Fatal("duas gerações não deveriam produzir o mesmo secret (fonte de aleatoriedade quebrada)")
	}
	if hash1 == hash2 {
		t.Fatal("hashes de secrets diferentes não deveriam colidir")
	}
	if hash1 == secret1 {
		t.Fatal("hash nunca deveria ser igual ao secret em texto claro")
	}
}

func TestHashAgentSecret_DeterministicoEVerificavel(t *testing.T) {
	secret, hash, err := GenerateAgentSecret()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if HashAgentSecret(secret) != hash {
		t.Fatal("HashAgentSecret(secret) deveria bater com o hash retornado por GenerateAgentSecret")
	}
	if HashAgentSecret("outro-valor-qualquer") == hash {
		t.Fatal("hash de um secret diferente não deveria colidir")
	}
}

func TestGenerateEnrollmentToken_UnicoENuncaTextoClaro(t *testing.T) {
	token1, hash1, err := GenerateEnrollmentToken()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	token2, hash2, err := GenerateEnrollmentToken()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if token1 == token2 {
		t.Fatal("duas gerações não deveriam produzir o mesmo token")
	}
	if hash1 == hash2 {
		t.Fatal("hashes de tokens diferentes não deveriam colidir")
	}
}

func TestHashEnrollmentToken_DeterministicoEVerificavel(t *testing.T) {
	token, hash, err := GenerateEnrollmentToken()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if HashEnrollmentToken(token) != hash {
		t.Fatal("HashEnrollmentToken(token) deveria bater com o hash retornado por GenerateEnrollmentToken")
	}
}
