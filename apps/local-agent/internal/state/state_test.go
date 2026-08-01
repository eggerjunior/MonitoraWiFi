package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ReturnsNilWhenFileDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	id, err := Load(path)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id != nil {
		t.Fatalf("esperava nil quando o arquivo não existe, recebeu %+v", id)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "agent.json")
	original := Identity{AgentID: "agent-1", AgentSecret: "s3cr3t", SiteID: "site-1"}

	if err := Save(path, original); err != nil {
		t.Fatalf("erro ao salvar: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("erro ao carregar: %v", err)
	}
	if loaded == nil || *loaded != original {
		t.Fatalf("esperava %+v, recebeu %+v", original, loaded)
	}
}

func TestSave_FilePermissionsAreRestricted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	if err := Save(path, Identity{AgentID: "a", AgentSecret: "b", SiteID: "c"}); err != nil {
		t.Fatalf("erro ao salvar: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("erro ao inspecionar arquivo: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("esperava permissão 0600, encontrou %o", perm)
	}
}
