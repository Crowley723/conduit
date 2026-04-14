package storage

import (
	"fmt"
	"github.com/Crowley723/conduit/internal/config"
)

func GetConnectionStringFromConfig(cfg *config.Config) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Storage.Host,
		cfg.Storage.Port,
		cfg.Storage.Username,
		cfg.Storage.Password,
		cfg.Storage.Database,
	)

	return connStr
}
