package commands

import (
	"fmt"
	"log/slog"

	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/server"
)

type ServeCommand struct {
	Port int `help:"Override port from config" short:"p"`
}

func (s *ServeCommand) Run(globals *Globals) error {
	logger, err := server.NewLogger(globals.LogLevel, globals.LogFormat)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	slog.SetDefault(logger)

	logger.Info("Starting conduit", "config", globals.Config, "log_level", globals.LogLevel, "log_format", globals.LogFormat)

	cfg, err := config.LoadConfig(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if s.Port != 0 {
		cfg.Server.Port = s.Port
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
