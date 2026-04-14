package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/Crowley723/conduit/internal/models"
	"github.com/jackc/pgx/v5"
)

// CreateDownloadToken stores a new one-time download token in the database.
func (p *DatabaseProvider) CreateDownloadToken(ctx context.Context, tokenHash string, certificateID int, principalIss, principalSub, passphrase string, expiresAt time.Time) error {
	query := `
		INSERT INTO certificate_download_tokens
			(token_hash, certificate_request_id, principal_iss, principal_sub, passphrase, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := p.pool.Exec(ctx, query, tokenHash, certificateID, principalIss, principalSub, passphrase, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create download token: %w", err)
	}

	return nil
}

// GetAndConsumeDownloadToken atomically marks the token as used and returns its data.
// Returns DownloadTokenNotFoundOrExpired if the token does not exist, is already used, or has expired.
func (p *DatabaseProvider) GetAndConsumeDownloadToken(ctx context.Context, tokenHash string) (*models.DownloadToken, error) {
	query := `
		UPDATE certificate_download_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING token_hash, certificate_request_id, principal_iss, principal_sub, passphrase, created_at, expires_at, used_at
	`

	var t models.DownloadToken
	err := p.pool.QueryRow(ctx, query, tokenHash).Scan(
		&t.TokenHash,
		&t.CertificateID,
		&t.PrincipalIss,
		&t.PrincipalSub,
		&t.Passphrase,
		&t.CreatedAt,
		&t.ExpiresAt,
		&t.UsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, DownloadTokenNotFoundOrExpired
		}
		return nil, fmt.Errorf("failed to consume download token: %w", err)
	}

	return &t, nil
}

// DeleteExpiredDownloadTokens removes tokens that expired more than 24 hours ago.
// Returns the number of rows deleted.
func (p *DatabaseProvider) DeleteExpiredDownloadTokens(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM certificate_download_tokens
		WHERE expires_at < NOW() - INTERVAL '24 hours'
	`

	result, err := p.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired download tokens: %w", err)
	}

	return result.RowsAffected(), nil
}
