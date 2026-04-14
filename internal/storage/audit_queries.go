package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/Crowley723/conduit/internal/models"

	"github.com/jackc/pgx/v5"
)

func (p *DatabaseProvider) InsertAuditLogCertificateDownload(ctx context.Context, certId int, sub, iss, ipAddress, rawUserAgent string) (*models.CertificateDownload, error) {
	query := `
		INSERT INTO certificate_downloads (certificate_request_id, downloader_sub, downloader_iss, ip_address, user_agent, downloaded_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var insertedId int
	err := p.pool.QueryRow(ctx, query, certId, sub, iss, ipAddress, rawUserAgent).Scan(&insertedId)
	if err != nil {
		return nil, fmt.Errorf("failed to insert download log: %w", err)
	}

	return p.GetCertificateDownloadAuditLogByID(ctx, insertedId)
}

func (p *DatabaseProvider) GetCertificateDownloadAuditLogByID(ctx context.Context, id int) (*models.CertificateDownload, error) {
	query := `
        SELECT id, certificate_request_id, downloader_sub, downloader_iss, ip_address, user_agent, downloaded_at
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
        SELECT id, certificate_request_id, downloader_sub, downloader_iss, ip_address, user_agent, downloaded_at
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
