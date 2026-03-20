package internal

import "fmt"

// NewDBConnector creates a new DBConnector based on the configuration driver.
func NewDBConnector(cfg Config) (DBConnector, error) {
	switch cfg.DBDriver {
	case "sqlite":
		return &SQLiteConnector{Path: cfg.DBPath}, nil
	case "postgres":
		return &PostgresConnector{
			Host:     cfg.DBHost,
			Port:     cfg.DBPort,
			User:     cfg.DBUser,
			Password: cfg.DBPassword,
			Name:     cfg.DBName,
			SSLMode:  cfg.DBSSLMode,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DBDriver)
	}
}
