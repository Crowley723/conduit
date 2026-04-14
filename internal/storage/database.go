package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/config"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*
var migrationsFS embed.FS

type DatabaseProvider struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewStorageProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	pPool, err := pgxpool.New(ctx, GetConnectionStringFromConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseProvider{pool: pPool, cfg: cfg}, nil
}

func (p *DatabaseProvider) GetPool() *pgxpool.Pool {
	return p.pool
}

func (p *DatabaseProvider) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

func (p *DatabaseProvider) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *DatabaseProvider) RunMigrations(ctx context.Context) error {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		    version VARCHAR(255) PRIMARY KEY,
		    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		    )
		`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			migrations = append(migrations, entry.Name())
		}
	}

	sort.Strings(migrations)

	for _, filename := range migrations {
		version := strings.Split(filename, "_")[0]

		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", version, err)
		}

		if exists {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		_, err = tx.Exec(ctx, string(content))
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		err = tx.Commit(ctx)
		if err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}
	}

	return nil
}

func (p *DatabaseProvider) EnsureSystemUser(ctx context.Context, logger *slog.Logger) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingIss, existingSub string
	err = tx.QueryRow(ctx, `
		SELECT iss, sub FROM users WHERE is_system = TRUE
	`).Scan(&existingIss, &existingSub)

	systemSub := "system"

	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO users (iss, sub, username, display_name, email, is_system, created_at)
			VALUES ($1, $2, $3, $4, $5, TRUE, NOW())
		`, p.cfg.Server.ExternalURL, systemSub, SystemUsername, SystemDisplayName, SystemEmail)

		if err != nil {
			return fmt.Errorf("failed to create system user: %w", err)
		}

		logger.Info("created system user", "iss", p.cfg.Server.ExternalURL, "sub", systemSub)

	} else if err != nil {
		return fmt.Errorf("failed to check for system user: %w", err)
	} else {
		if existingIss != p.cfg.Server.ExternalURL {
			logger.Warn("external URL changed, updating system user",
				"old_iss", existingIss,
				"new_iss", p.cfg.Server.ExternalURL,
			)

			// Update the system user's iss
			_, err = tx.Exec(ctx, `
				UPDATE users 
				SET iss = $1, username = $2, display_name = $3, email = $4
				WHERE is_system = TRUE
			`, p.cfg.Server.ExternalURL, SystemUsername, SystemDisplayName, SystemEmail)

			if err != nil {
				return fmt.Errorf("failed to update system user: %w", err)
			}

			_, err = tx.Exec(ctx, `
				UPDATE certificate_requests 
				SET owner_iss = $1 
				WHERE owner_iss = $2 AND owner_sub = $3
			`, p.cfg.Server.ExternalURL, existingIss, systemSub)

			if err != nil {
				return fmt.Errorf("failed to update certificate_requests: %w", err)
			}

			// Update all references in certificate_events (requester)
			_, err = tx.Exec(ctx, `
				UPDATE certificate_events 
				SET requester_iss = $1 
				WHERE requester_iss = $2 AND requester_sub = $3
			`, p.cfg.Server.ExternalURL, existingIss, systemSub)

			if err != nil {
				return fmt.Errorf("failed to update certificate_events requester: %w", err)
			}

			// Update all references in certificate_events (reviewer)
			_, err = tx.Exec(ctx, `
				UPDATE certificate_events 
				SET reviewer_iss = $1 
				WHERE reviewer_iss = $2 AND reviewer_sub = $3
			`, p.cfg.Server.ExternalURL, existingIss, systemSub)

			if err != nil {
				return fmt.Errorf("failed to update certificate_events reviewer: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (p *DatabaseProvider) GetSystemUser(ctx context.Context) (iss, sub string, err error) {
	err = p.pool.QueryRow(ctx, `
		SELECT iss, sub FROM users WHERE is_system = TRUE
	`).Scan(&iss, &sub)

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("system user not found")
	}

	return iss, sub, err
}
