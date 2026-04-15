package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/Crowley723/conduit/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*
var migrationsFS embed.FS

const latestMigrationVersion = 1

type DatabaseProvider struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewStorageProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	pPool, err := pgxpool.New(ctx, GetConnectionStringFromConfig(cfg))
	if err != nil {
		return nil, simplifyDatabaseError(err)
	}

	if err := pPool.Ping(ctx); err != nil {
		return nil, simplifyDatabaseError(err)
	}

	return &DatabaseProvider{pool: pPool, cfg: cfg}, nil
}

func simplifyDatabaseError(err error) error {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Op == "dial" {
			if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
				return fmt.Errorf("failed to connect to %s: connection refused", netErr.Addr)
			}
		}
		return fmt.Errorf("failed to connect to %s: %v", netErr.Addr, netErr.Err)
	}

	// For other errors, try to extract just the essential message
	errMsg := err.Error()
	if strings.Contains(errMsg, "dial error:") {
		parts := strings.Split(errMsg, "dial error: ")
		if len(parts) > 1 {
			return fmt.Errorf("failed to connect to database: %s", parts[len(parts)-1])
		}
	}

	return fmt.Errorf("failed to connect to database: %w", err)
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

func (p *DatabaseProvider) GetCurrentMigrationVersion(ctx context.Context) (int, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return -1, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	var version int
	err = conn.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return -1, fmt.Errorf("failed to get current migration version: %w", err)
	}

	return version, nil
}

func (p *DatabaseProvider) RunMigrations(ctx context.Context) error {
	return p.RunUpMigrations(ctx, latestMigrationVersion)
}

func (p *DatabaseProvider) RunUpMigrations(ctx context.Context, targetVersion int) error {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		    version INTEGER PRIMARY KEY,
		    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		    )
		`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	currentVersion, err := p.GetCurrentMigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if targetVersion <= currentVersion {
		return fmt.Errorf("target version %d is not ahead of current version %d", targetVersion, currentVersion)
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
		version, err := strconv.Atoi(strings.Split(filename, "_")[0])
		if err != nil {
			return fmt.Errorf("failed to parse migration version from %s: %w", filename, err)
		}

		if version <= currentVersion {
			continue
		}

		if version > targetVersion {
			break
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

func (p *DatabaseProvider) RunDownMigrations(ctx context.Context, targetVersion int) error {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	currentVersion, err := p.GetCurrentMigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version %d is not below current version %d", targetVersion, currentVersion)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".down.sql") {
			migrations = append(migrations, entry.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(migrations)))

	for _, filename := range migrations {
		version, err := strconv.Atoi(strings.Split(filename, "_")[0])
		if err != nil {
			return fmt.Errorf("failed to parse migration version from %s: %w", filename, err)
		}

		if version > currentVersion {
			continue
		}

		if version <= targetVersion {
			break
		}

		content, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", filename, err)
		}

		_, err = tx.Exec(ctx, string(content))
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		_, err = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", version)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to remove migration record %s: %w", filename, err)
		}

		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
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

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.Exec(ctx, `
			INSERT INTO users (iss, sub, username, display_name, email, is_system, created_at)
			VALUES ($1, $2, $3, $4, $5, TRUE, NOW())
		`, p.cfg.Server.ExternalURL, systemSub, SystemUsername, SystemDisplayName, SystemEmail)

			if err != nil {
				return fmt.Errorf("failed to create system user: %w", err)
			}

			logger.Info("created system user", "iss", p.cfg.Server.ExternalURL, "sub", systemSub)

		} else {
			return fmt.Errorf("failed to check for system user: %w", err)
		}
	} else {
		if existingIss != p.cfg.Server.ExternalURL {
			logger.Warn("external URL changed, updating system user",
				"old_iss", existingIss,
				"new_iss", p.cfg.Server.ExternalURL,
			)

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

			_, err = tx.Exec(ctx, `
				UPDATE certificate_events 
				SET requester_iss = $1 
				WHERE requester_iss = $2 AND requester_sub = $3
			`, p.cfg.Server.ExternalURL, existingIss, systemSub)

			if err != nil {
				return fmt.Errorf("failed to update certificate_events requester: %w", err)
			}

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
