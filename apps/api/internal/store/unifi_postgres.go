package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUniFiDevices struct{ Pool *pgxpool.Pool }
type PostgresUniFiClients struct{ Pool *pgxpool.Pool }

func (s *PostgresUniFiDevices) ReplaceBySite(ctx context.Context, siteID uuid.UUID, devices []UniFiDevice) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM unifi_devices WHERE site_id = $1`, siteID); err != nil {
		return err
	}

	if len(devices) > 0 {
		batch := &pgx.Batch{}
		for _, d := range devices {
			features, _ := json.Marshal(d.Features)
			interfaces, _ := json.Marshal(d.Interfaces)
			batch.Queue(
				`INSERT INTO unifi_devices (site_id, external_id, mac_address, ip_address, name, model, state, firmware_version, features, interfaces)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				siteID, d.ExternalID, d.MACAddress, d.IPAddress, d.Name, d.Model, d.State, d.FirmwareVersion, features, interfaces)
		}
		br := tx.SendBatch(ctx, batch)
		for range devices {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresUniFiDevices) ListBySite(ctx context.Context, siteID uuid.UUID) ([]UniFiDevice, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, external_id, mac_address, ip_address, name, model, state, firmware_version, features, interfaces
		 FROM unifi_devices WHERE site_id = $1 ORDER BY name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UniFiDevice
	for rows.Next() {
		var d UniFiDevice
		var features, interfaces []byte
		if err := rows.Scan(&d.ID, &d.SiteID, &d.ExternalID, &d.MACAddress, &d.IPAddress, &d.Name, &d.Model, &d.State, &d.FirmwareVersion, &features, &interfaces); err != nil {
			return nil, err
		}
		json.Unmarshal(features, &d.Features)
		json.Unmarshal(interfaces, &d.Interfaces)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *PostgresUniFiClients) ReplaceBySite(ctx context.Context, siteID uuid.UUID, clients []UniFiClient) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM unifi_clients WHERE site_id = $1`, siteID); err != nil {
		return err
	}

	if len(clients) > 0 {
		batch := &pgx.Batch{}
		for _, c := range clients {
			batch.Queue(
				`INSERT INTO unifi_clients (site_id, external_id, type, name, ip_address, mac_address, connected_at, uplink_device_id)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				siteID, c.ExternalID, c.Type, c.Name, c.IPAddress, c.MACAddress, c.ConnectedAt, c.UplinkDeviceID)
		}
		br := tx.SendBatch(ctx, batch)
		for range clients {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresUniFiClients) ListBySite(ctx context.Context, siteID uuid.UUID) ([]UniFiClient, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, external_id, type, name, ip_address, mac_address, connected_at, uplink_device_id
		 FROM unifi_clients WHERE site_id = $1 ORDER BY name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UniFiClient
	for rows.Next() {
		var c UniFiClient
		if err := rows.Scan(&c.ID, &c.SiteID, &c.ExternalID, &c.Type, &c.Name, &c.IPAddress, &c.MACAddress, &c.ConnectedAt, &c.UplinkDeviceID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
