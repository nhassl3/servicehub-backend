package cmd

import (
	"log"
	"os"

	"github.com/nhassl3/servicehub-backend/internal/config"
	"github.com/nhassl3/servicehub-backend/pkg/logger"
	"go.uber.org/zap"
)

// MustLoadLogger inits logger zap with panic if it's occurred
func MustLoadLogger(level string) *zap.Logger {
	zapLogger, err := logger.NewZapLogger(level)
	if err != nil {
		log.Fatalf("main: init logger: %s", err) // os.Exit(1)
		return nil
	}
	return zapLogger
}

// MustLoadConfig loads configuration of the project.
// If error occurred when loading cfg - Fatal log with error message
func MustLoadConfig() *config.Config {
	// Config file: public settings (ports, DB host, log level, etc.)
	// Controlled by CONFIG_FILE env var; defaults to environment-aware path.
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		env := os.Getenv("ENVIRONMENT")
		switch env {
		case "prod":
			configFile = "config/prod.yaml"
		case "dev":
			configFile = "config/dev.yaml"
		default:
			configFile = "config/local.yaml"
		}
	}

	// Env file: secrets (DB password, PASETO key, etc.)
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}

	cfg, err := config.Load(configFile, envFile)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	return cfg
}
