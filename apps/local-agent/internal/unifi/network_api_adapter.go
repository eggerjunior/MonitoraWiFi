package unifi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NetworkAPIAdapter fala com a Network API local (UniFi Network 10.x+),
// autenticada via header X-API-KEY (confirmado em
// docs/unifi/verificacoes-pendentes-instalacao.md, itens 3/4). O console usa
// certificado autoassinado por padrão nesta instalação — InsecureSkipVerify
// é aceitável aqui porque a chamada nunca sai da LAN (o agente já está
// dentro dela, ADR-001); não seria aceitável se esta chamada cruzasse a
// internet.
type NetworkAPIAdapter struct {
	baseURL    string // ex.: "https://192.168.110.1"
	apiKey     string
	httpClient *http.Client
}

func NewNetworkAPIAdapter(baseURL, apiKey string) *NetworkAPIAdapter {
	return &NetworkAPIAdapter{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // console LAN com cert autoassinado, ver comentário acima
			},
		},
	}
}

type listEnvelope[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T `json:"data"`
}

type siteDTO struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"`
	Name              string `json:"name"`
}

type deviceDTO struct {
	ID                string   `json:"id"`
	MACAddress        string   `json:"macAddress"`
	IPAddress         string   `json:"ipAddress"`
	Name              string   `json:"name"`
	Model             string   `json:"model"`
	State             string   `json:"state"`
	FirmwareVersion   string   `json:"firmwareVersion"`
	FirmwareUpdatable bool     `json:"firmwareUpdatable"`
	Features          []string `json:"features"`
	Interfaces        []string `json:"interfaces"`
}

// deviceDetailDTO espelha só o campo que precisamos da resposta de
// detalhe de um device (`GET .../devices/{id}`) — bem mais rica que a de
// lista (confirmado em 2026-08-02: a lista não traz `uplink`, `features`
// e `interfaces` vêm em formatos diferentes na uma e na outra). Extrair só
// o uplink aqui evita duplicar o parsing de radios/portas, que é escopo
// de outra funcionalidade (capability-matrix itens 6/7, ainda não
// consumidos pelo produto).
type deviceDetailDTO struct {
	Uplink *struct {
		DeviceID string `json:"deviceId"`
	} `json:"uplink"`
}

type clientDTO struct {
	Type           string `json:"type"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	ConnectedAt    string `json:"connectedAt"`
	IPAddress      string `json:"ipAddress"`
	MACAddress     string `json:"macAddress"`
	UplinkDeviceID string `json:"uplinkDeviceId"`
}

func (a *NetworkAPIAdapter) ListSites(ctx context.Context) ([]Site, error) {
	var env listEnvelope[siteDTO]
	if err := a.getJSON(ctx, "/proxy/network/integration/v1/sites", &env); err != nil {
		return nil, err
	}
	out := make([]Site, 0, len(env.Data))
	for _, d := range env.Data {
		out = append(out, Site{ID: d.ID, InternalReference: d.InternalReference, Name: d.Name})
	}
	return out, nil
}

// ListDevices busca a lista e, pra cada device, o detalhe (chamada extra
// por device — só assim dá pra obter `uplink.deviceId`, confirmado
// ausente na resposta de lista). Uma falha ao buscar o detalhe de um
// device não derruba a sincronização inteira — só aquele device fica sem
// uplink nesse ciclo (recuperável no próximo).
func (a *NetworkAPIAdapter) ListDevices(ctx context.Context, siteID string) ([]Device, error) {
	var env listEnvelope[deviceDTO]
	path := fmt.Sprintf("/proxy/network/integration/v1/sites/%s/devices", siteID)
	if err := a.getJSON(ctx, path, &env); err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(env.Data))
	for _, d := range env.Data {
		device := Device{
			ID:                d.ID,
			MACAddress:        d.MACAddress,
			IPAddress:         d.IPAddress,
			Name:              d.Name,
			Model:             d.Model,
			State:             d.State,
			FirmwareVersion:   d.FirmwareVersion,
			FirmwareUpdatable: d.FirmwareUpdatable,
			Features:          d.Features,
			Interfaces:        d.Interfaces,
		}
		if uplink, err := a.getDeviceUplink(ctx, siteID, d.ID); err == nil {
			device.UplinkDeviceID = uplink
		}
		out = append(out, device)
	}
	return out, nil
}

func (a *NetworkAPIAdapter) getDeviceUplink(ctx context.Context, siteID, deviceID string) (string, error) {
	var detail deviceDetailDTO
	path := fmt.Sprintf("/proxy/network/integration/v1/sites/%s/devices/%s", siteID, deviceID)
	if err := a.getJSON(ctx, path, &detail); err != nil {
		return "", err
	}
	if detail.Uplink == nil {
		return "", nil
	}
	return detail.Uplink.DeviceID, nil
}

// ListClients busca todas as páginas — a API pagina em blocos de 25 por
// padrão (confirmado: totalCount 80, count 25 na primeira chamada real) e
// nunca deve reportar só a primeira página como se fosse o total.
func (a *NetworkAPIAdapter) ListClients(ctx context.Context, siteID string) ([]Client, error) {
	var out []Client
	offset := 0
	const limit = 200

	for {
		var env listEnvelope[clientDTO]
		path := fmt.Sprintf("/proxy/network/integration/v1/sites/%s/clients?offset=%d&limit=%d", siteID, offset, limit)
		if err := a.getJSON(ctx, path, &env); err != nil {
			return nil, err
		}
		for _, d := range env.Data {
			connectedAt, _ := time.Parse(time.RFC3339, d.ConnectedAt)
			out = append(out, Client{
				ID:             d.ID,
				Type:           d.Type,
				Name:           d.Name,
				IPAddress:      d.IPAddress,
				MACAddress:     d.MACAddress,
				ConnectedAt:    connectedAt,
				UplinkDeviceID: d.UplinkDeviceID,
			})
		}
		offset += len(env.Data)
		if len(env.Data) == 0 || offset >= env.TotalCount {
			break
		}
	}
	return out, nil
}

func (a *NetworkAPIAdapter) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("criar requisição: %w", err)
	}
	req.Header.Set("X-API-KEY", a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requisição à Network API local: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Network API local retornou status %d para %s", resp.StatusCode, path)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decodificar resposta da Network API local: %w", err)
	}
	return nil
}
