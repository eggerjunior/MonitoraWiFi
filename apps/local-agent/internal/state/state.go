// Package state persiste a identidade do agente (agent_id + agent_secret)
// localmente, para sobreviver a reinícios do processo sem precisar enrolar
// de novo. Arquivo com permissão 0600 — é a credencial de longa duração do
// agente (Seção 2.2: "Armazenamento seguro").
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Identity struct {
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
	SiteID      string `json:"site_id"`
}

// Load retorna (nil, nil) quando o arquivo ainda não existe — condição
// normal no primeiro start, não um erro.
func Load(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return nil, err
	}
	return &id, nil
}

func Save(path string, id Identity) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
