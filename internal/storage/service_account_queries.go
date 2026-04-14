package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/models"

	"github.com/jackc/pgx/v5"
)

func (p *DatabaseProvider) CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO service_accounts (sub, iss, name, lookup_id, token_hash, token_expires_at, is_disabled, created_by_sub, created_by_iss, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
    `
	result, err := tx.Exec(ctx, query, serviceAccount.Sub, serviceAccount.Iss, serviceAccount.Name, serviceAccount.LookupId, serviceAccount.SecretHash, serviceAccount.TokenExpiresAt, serviceAccount.IsDisabled, serviceAccount.CreatedBySub, serviceAccount.CreatedByIss)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	if result.RowsAffected() != 1 {
		return nil, fmt.Errorf("failed to create service account: expected 1 row, got %d", result.RowsAffected())
	}

	scopesQuery := `
        INSERT INTO service_account_scopes (owner_sub, owner_iss, scope_name)
        VALUES ($1, $2, $3)
    `
	for _, scope := range serviceAccount.Scopes {
		_, err := tx.Exec(ctx, scopesQuery, serviceAccount.Sub, serviceAccount.Iss, scope)
		if err != nil {
			return nil, fmt.Errorf("failed to insert scope %s: %w", scope, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return p.GetServiceAccountByID(ctx, serviceAccount.Iss, serviceAccount.Sub)
}

func (p *DatabaseProvider) GetServiceAccountByID(ctx context.Context, iss, sub string) (*models.ServiceAccount, error) {
	query := `
		SELECT iss, sub, name, lookup_id, token_hash, token_expires_at, is_disabled, deleted_at, created_by_sub, created_by_iss, created_at
		FROM service_accounts
		WHERE iss = $1 AND sub = $2
	`

	scopesQuery := `
		SELECT scope_name
		FROM service_account_scopes
		WHERE owner_iss = $1 AND owner_sub = $2
	`

	var serviceAccount models.ServiceAccount
	err := p.pool.QueryRow(ctx, query, iss, sub).Scan(
		&serviceAccount.Iss,
		&serviceAccount.Sub,
		&serviceAccount.Name,
		&serviceAccount.LookupId,
		&serviceAccount.SecretHash,
		&serviceAccount.TokenExpiresAt,
		&serviceAccount.IsDisabled,
		&serviceAccount.DeletedAt,
		&serviceAccount.CreatedBySub,
		&serviceAccount.CreatedByIss,
		&serviceAccount.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("service account not found")
		}
		return nil, fmt.Errorf("failed to get service account by id: %w", err)
	}

	rows, err := p.pool.Query(ctx, scopesQuery, iss, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes for service account '%s:%s': %w", serviceAccount.Sub, serviceAccount.Iss, err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scopeName string
		if err := rows.Scan(&scopeName); err != nil {
			return nil, fmt.Errorf("failed to scan scopes name: %w", err)
		}
		scopes = append(scopes, scopeName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate scopes: %w", err)
	}

	serviceAccount.Scopes = scopes
	if serviceAccount.Scopes == nil {
		serviceAccount.Scopes = []string{}
	}

	return &serviceAccount, nil
}

func (p *DatabaseProvider) GetServiceAccountByLookupId(ctx context.Context, lookupId string) (*models.ServiceAccount, error) {
	query := `
		SELECT iss, sub, name, lookup_id, token_hash, token_expires_at, is_disabled, deleted_at, created_by_sub, created_by_iss, created_at
		FROM service_accounts
		where lookup_id = $1
	`

	scopesQuery := `
		SELECT scope_name
        FROM service_account_scopes
        WHERE owner_iss = $1 AND owner_sub = $2`

	var serviceAccount models.ServiceAccount
	err := p.pool.QueryRow(ctx, query, lookupId).Scan(
		&serviceAccount.Iss,
		&serviceAccount.Sub,
		&serviceAccount.Name,
		&serviceAccount.LookupId,
		&serviceAccount.SecretHash,
		&serviceAccount.TokenExpiresAt,
		&serviceAccount.IsDisabled,
		&serviceAccount.DeletedAt,
		&serviceAccount.CreatedBySub,
		&serviceAccount.CreatedByIss,
		&serviceAccount.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("service account not found")
		}
		return nil, fmt.Errorf("failed to get service account by id: %w", err)
	}

	rows, err := p.pool.Query(ctx, scopesQuery, serviceAccount.Iss, serviceAccount.Sub)
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes for service account '%s:%s': %w", serviceAccount.Sub, serviceAccount.Iss, err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scopeName string
		if err := rows.Scan(&scopeName); err != nil {
			return nil, fmt.Errorf("failed to scan scopes name: %w", err)
		}
		scopes = append(scopes, scopeName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate scopes: %w", err)
	}

	serviceAccount.Scopes = scopes
	if serviceAccount.Scopes == nil {
		serviceAccount.Scopes = []string{}
	}

	return &serviceAccount, nil
}

func (p *DatabaseProvider) GetServiceAccountsByCreator(ctx context.Context, iss, sub string) ([]*models.ServiceAccount, error) {
	query := `
       SELECT iss, sub, name, lookup_id, token_hash, token_expires_at, is_disabled, deleted_at, created_by_sub, created_by_iss, created_at
       FROM service_accounts
       WHERE created_by_iss = $1 AND created_by_sub = $2
       ORDER BY created_at DESC`

	scopesQuery := `
       SELECT owner_iss, owner_sub, scope_name
       FROM service_account_scopes
       WHERE (owner_iss, owner_sub) IN (SELECT UNNEST($1::text[]), UNNEST($2::text[]))
       ORDER BY owner_iss, owner_sub`

	rows, err := p.pool.Query(ctx, query, iss, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to get service accounts for creator '%s/%s': %w", iss, sub, err)
	}
	defer rows.Close()

	var serviceAccounts []*models.ServiceAccount
	var issValues []string
	var subValues []string

	for rows.Next() {
		var serviceAccount models.ServiceAccount
		if err := rows.Scan(
			&serviceAccount.Iss,
			&serviceAccount.Sub,
			&serviceAccount.Name,
			&serviceAccount.LookupId,
			&serviceAccount.SecretHash,
			&serviceAccount.TokenExpiresAt,
			&serviceAccount.IsDisabled,
			&serviceAccount.DeletedAt,
			&serviceAccount.CreatedBySub,
			&serviceAccount.CreatedByIss,
			&serviceAccount.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan service account: %w", err)
		}
		serviceAccount.Scopes = []string{}
		serviceAccounts = append(serviceAccounts, &serviceAccount)
		issValues = append(issValues, serviceAccount.Iss)
		subValues = append(subValues, serviceAccount.Sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate service accounts: %w", err)
	}

	if len(serviceAccounts) == 0 {
		return serviceAccounts, nil
	}

	scopeRows, err := p.pool.Query(ctx, scopesQuery, issValues, subValues)
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes for service accounts: %w", err)
	}
	defer scopeRows.Close()

	saMap := make(map[string]*models.ServiceAccount)
	for _, sa := range serviceAccounts {
		key := sa.Iss + ":" + sa.Sub
		saMap[key] = sa
	}

	for scopeRows.Next() {
		var ownerIss, ownerSub, scopeName string
		if err := scopeRows.Scan(&ownerIss, &ownerSub, &scopeName); err != nil {
			return nil, fmt.Errorf("failed to scan scope: %w", err)
		}

		key := ownerIss + ":" + ownerSub
		if sa, ok := saMap[key]; ok {
			sa.Scopes = append(sa.Scopes, scopeName)
		}
	}

	if err := scopeRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate scopes: %w", err)
	}

	return serviceAccounts, nil
}

func (p *DatabaseProvider) PauseServiceAccount(ctx context.Context, iss, sub string) error {
	query := `
       UPDATE service_accounts
       SET is_disabled = TRUE
       WHERE iss = $1 AND sub = $2 AND deleted_at IS NULL`

	result, err := p.pool.Exec(ctx, query, iss, sub)
	if err != nil {
		return fmt.Errorf("failed to pause service account: %w", err)
	}

	if result.RowsAffected() < 1 {
		return fmt.Errorf("failed to pause service account: no rows updated")
	}

	if result.RowsAffected() > 1 {
		return fmt.Errorf("failed to pause service account: multiple rows updated")
	}

	return nil
}

func (p *DatabaseProvider) UnpauseServiceAccount(ctx context.Context, iss, sub string) error {
	query := `
       UPDATE service_accounts
       SET is_disabled = FALSE
       WHERE iss = $1 AND sub = $2 AND deleted_at IS NULL`

	result, err := p.pool.Exec(ctx, query, iss, sub)
	if err != nil {
		return fmt.Errorf("failed to unpause service account: %w", err)
	}

	if result.RowsAffected() < 1 {
		return fmt.Errorf("failed to unpause service account: no rows updated")
	}

	if result.RowsAffected() > 1 {
		return fmt.Errorf("failed to unpause service account: multiple rows updated")
	}

	return nil
}

func (p *DatabaseProvider) DeleteServiceAccount(ctx context.Context, iss, sub string) error {
	query := `
       UPDATE service_accounts
       SET deleted_at = CURRENT_TIMESTAMP, is_disabled = TRUE
       WHERE iss = $1 AND sub = $2 AND deleted_at IS NULL`

	result, err := p.pool.Exec(ctx, query, iss, sub)
	if err != nil {
		return fmt.Errorf("failed to delete service account: %w", err)
	}

	if result.RowsAffected() < 1 {
		return fmt.Errorf("failed to delete service account: no rows updated or already deleted")
	}

	if result.RowsAffected() > 1 {
		return fmt.Errorf("failed to delete service account: multiple rows updated")
	}

	return nil
}

// Legacy aliases for backwards compatibility
func (p *DatabaseProvider) DisableServiceAccount(ctx context.Context, iss, sub string) error {
	return p.PauseServiceAccount(ctx, iss, sub)
}

func (p *DatabaseProvider) EnableServiceAccount(ctx context.Context, iss, sub string) error {
	return p.UnpauseServiceAccount(ctx, iss, sub)
}
