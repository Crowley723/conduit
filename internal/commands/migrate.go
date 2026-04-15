package commands

import (
	"context"
	"fmt"

	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/storage"
)

type MigrateCommand struct {
	Up   MigrateUpCommand   `cmd:"" help:"Upgrade migration version"`
	Down MigrateDownCommand `cmd:"" help:"Downgrade migration version"`
}

type MigrateUpCommand struct {
	TargetMigration int `short:"t" required:"" help:"Target migration version"`
}

type MigrateDownCommand struct {
	TargetMigration int `short:"t" required:"" help:"Target migration version"`
}

func (cmd *MigrateUpCommand) Run(globals *Globals) error {
	cfg, err := config.LoadConfig(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()
	provider, err := storage.NewStorageProvider(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage provider: %w", err)
	}
	defer provider.Close()

	err = provider.RunUpMigrations(ctx, cmd.TargetMigration)
	if err != nil {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}

func (cmd *MigrateDownCommand) Run(globals *Globals) error {
	cfg, err := config.LoadConfig(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()
	provider, err := storage.NewStorageProvider(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage provider: %w", err)
	}
	defer provider.Close()

	err = provider.RunDownMigrations(ctx, cmd.TargetMigration)
	if err != nil {
		return fmt.Errorf("failed to run down migrations: %w", err)
	}

	return nil
}
