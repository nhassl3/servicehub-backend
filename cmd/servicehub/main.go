package main

import (
	"log"
	"os"

	"github.com/nhassl3/servicehub/internal/app"
	"github.com/nhassl3/servicehub/internal/config"
)

func main() {
	// Config file: public settings (ports, DB host, log level, etc.)
	// Controlled by CONFIG_FILE env var; defaults to environment-aware path.
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		env := os.Getenv("ENVIRONMENT")
		switch env {
		case "prod":
			configFile = "config/prod.yaml"
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

	if err := app.Run(cfg); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
