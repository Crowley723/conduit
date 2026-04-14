package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/models"

	"github.com/jackc/pgx/v5"
)

// CreateUser adds a user to the database.
func (p *DatabaseProvider) CreateUser(ctx context.Context, sub, iss, username, displayName, email string) (*models.User, error) {
	query := `
		INSERT INTO users (sub, iss, username, display_name, email, last_logged_in, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	result, err := p.pool.Exec(ctx, query, sub, iss, username, displayName, email)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	if result.RowsAffected() < 1 {
		return nil, fmt.Errorf("failed to create user: no rows inserted")
	}

	if result.RowsAffected() > 1 {
		return nil, fmt.Errorf("failed to create user: multiple rows inserted")
	}

	return p.GetUserByID(ctx, iss, sub)
}

// UpsertUser adds a user to the database, or updates the existing user, including their groups.
func (p *DatabaseProvider) UpsertUser(ctx context.Context, sub, iss, username, displayName, email string, groups []string) (*models.User, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	userQuery := `
        INSERT INTO users (sub, iss, username, display_name, email, last_logged_in, created_at)
        VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        ON CONFLICT (iss, sub) 
        DO UPDATE SET 
            username = EXCLUDED.username,
            display_name = EXCLUDED.display_name,
            email = EXCLUDED.email,
            last_logged_in = CURRENT_TIMESTAMP
    `

	_, err = tx.Exec(ctx, userQuery, sub, iss, username, displayName, email)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	deleteGroupsQuery := `
        DELETE FROM user_groups 
        WHERE owner_iss = $1 AND owner_sub = $2
    `
	_, err = tx.Exec(ctx, deleteGroupsQuery, iss, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old groups: %w", err)
	}

	if len(groups) > 0 {
		insertGroupsQuery := `
            INSERT INTO user_groups (owner_iss, owner_sub, group_name)
            VALUES ($1, $2, unnest($3::text[]))
        `
		_, err = tx.Exec(ctx, insertGroupsQuery, iss, sub, groups)
		if err != nil {
			return nil, fmt.Errorf("failed to insert groups: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return p.GetUserByID(ctx, iss, sub)
}

// GetUserByID returns a user, including groups, given their iss and sub claims.
func (p *DatabaseProvider) GetUserByID(ctx context.Context, iss, sub string) (*models.User, error) {
	query := `
		SELECT iss, sub, username, display_name, email, is_system, last_logged_in, created_at
		FROM users
		WHERE iss = $1 AND sub = $2
	`

	groupsQuery := `
		SELECT group_name 
        FROM user_groups
        WHERE owner_iss = $1 AND owner_sub = $2`

	var user models.User
	err := p.pool.QueryRow(ctx, query, iss, sub).Scan(
		&user.Iss,
		&user.Sub,
		&user.Username,
		&user.DisplayName,
		&user.Email,
		&user.IsSystem,
		&user.LastLoggedIn,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	rows, err := p.pool.Query(ctx, groupsQuery, user.Iss, user.Sub)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups for user '%s:%s': %w", user.Sub, user.Iss, err)
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var groupName string
		if err := rows.Scan(&groupName); err != nil {
			return nil, fmt.Errorf("failed to scan group name: %w", err)
		}
		groups = append(groups, groupName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate groups: %w", err)
	}

	user.Groups = groups
	if user.Groups == nil {
		user.Groups = []string{}
	}

	return &user, nil
}
