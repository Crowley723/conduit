package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/utils"

	"github.com/avct/uasurfer"
	"github.com/jackc/pgx/v5"
)

func (p *DatabaseProvider) InsertAuditLogCertificateDownload(ctx context.Context, certId int, sub, iss, ipAddress, rawUserAgent string, userAgent uasurfer.UserAgent) (*models.CertificateDownload, error) {
	query := `
		INSERT INTO certificate_downloads (certificate_request_id, downloader_sub, downloader_iss, ip_address, user_agent, browser_name, browser_version, os_name, os_version, device_type, downloaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var insertedId int
	err := p.pool.QueryRow(ctx, query,
		certId, sub, iss, ipAddress, rawUserAgent,
		userAgent.Browser.Name.String(),
		utils.UserAgentVersionToString(userAgent.Browser.Version),
		userAgent.OS.Name.String(),
		utils.UserAgentVersionToString(userAgent.OS.Version),
		userAgent.DeviceType.String(),
	).Scan(&insertedId)
	if err != nil {
		return nil, fmt.Errorf("failed to insert download log: %w", err)
	}

	return p.GetCertificateDownloadAuditLogByID(ctx, insertedId)
}

// GetCertificateDownloadAuditLogByID returns a single download log entry by its ID
func (p *DatabaseProvider) GetCertificateDownloadAuditLogByID(ctx context.Context, id int) (*models.CertificateDownload, error) {
	query := `
        SELECT id, certificate_request_id, downloader_sub, downloader_iss, ip_address, user_agent,
               browser_name, browser_version, os_name, os_version, device_type, downloaded_at
        FROM certificate_downloads
        WHERE id = $1
    `

	var d models.CertificateDownload
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.CertificateRequestID,
		&d.Sub,
		&d.Iss,
		&d.IPAddress,
		&d.UserAgent,
		&d.BrowserName,
		&d.BrowserVersion,
		&d.OSName,
		&d.OSVersion,
		&d.DeviceType,
		&d.DownloadedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("download log not found")
		}
		return nil, fmt.Errorf("failed to get download log: %w", err)
	}

	return &d, nil
}

func (p *DatabaseProvider) GetRecentCertificateDownloadLogs(ctx context.Context, limit int) ([]models.CertificateDownload, error) {
	query := `
        SELECT id, certificate_request_id, downloader_sub, downloader_iss, ip_address, user_agent,
               browser_name, browser_version, os_name, os_version, device_type, downloaded_at
        FROM certificate_downloads
        ORDER BY downloaded_at DESC
        LIMIT $1
    `

	rows, err := p.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent download logs: %w", err)
	}
	defer rows.Close()

	var downloads []models.CertificateDownload
	for rows.Next() {
		var d models.CertificateDownload
		err := rows.Scan(
			&d.ID,
			&d.CertificateRequestID,
			&d.Sub,
			&d.Iss,
			&d.IPAddress,
			&d.UserAgent,
			&d.BrowserName,
			&d.BrowserVersion,
			&d.OSName,
			&d.OSVersion,
			&d.DeviceType,
			&d.DownloadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan download log: %w", err)
		}
		downloads = append(downloads, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate download logs: %w", err)
	}

	return downloads, nil
}

// CreateWhitelistEvent creates an audit event for a whitelist entry
func (p *DatabaseProvider) CreateWhitelistEvent(ctx context.Context, whitelistID int, actorIss, actorSub, eventType, notes string, clientIP, userAgent *string) error {
	query := `
        INSERT INTO firewall_whitelist_events (whitelist_id, actor_iss, actor_sub, event_type, notes, client_ip, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err := p.pool.Exec(ctx, query, whitelistID, actorIss, actorSub, eventType, notes, clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("failed to create whitelist event: %w", err)
	}

	return nil
}

// GetWhitelistEventsByEntry gets all audit events for a specific whitelist entry
func (p *DatabaseProvider) GetWhitelistEventsByEntry(ctx context.Context, whitelistID int) ([]*models.FirewallIPWhitelistEvent, error) {
	query := `
        SELECT fwe.id, fwe.whitelist_id, fwe.actor_iss, fwe.actor_sub, fwe.event_type, fwe.notes, fwe.client_ip::text,
               fwe.user_agent, fwe.created_at,
               COALESCE(actor.username, sa_creator.username) as actor_username,
               COALESCE(actor.display_name, sa_creator.display_name) as actor_display_name
        FROM firewall_whitelist_events fwe
        LEFT JOIN users actor ON fwe.actor_iss = actor.iss AND fwe.actor_sub = actor.sub
        LEFT JOIN service_accounts sa ON fwe.actor_iss = sa.iss AND fwe.actor_sub = sa.sub
        LEFT JOIN users sa_creator ON sa.created_by_iss = sa_creator.iss AND sa.created_by_sub = sa_creator.sub
        WHERE fwe.whitelist_id = $1
        ORDER BY fwe.created_at DESC
    `

	rows, err := p.pool.Query(ctx, query, whitelistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelist events: %w", err)
	}
	defer rows.Close()

	var events []*models.FirewallIPWhitelistEvent
	for rows.Next() {
		var event models.FirewallIPWhitelistEvent
		err := rows.Scan(
			&event.ID,
			&event.WhitelistID,
			&event.ActorISS,
			&event.ActorSub,
			&event.EventType,
			&event.Notes,
			&event.ClientIP,
			&event.UserAgent,
			&event.CreatedAt,
			&event.ActorUsername,
			&event.ActorDisplayName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan whitelist event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate whitelist events: %w", err)
	}

	return events, nil
}
