package main

import (
	"log"

	"github.com/nhassl3/servicehub-backend/cmd"
	"github.com/nhassl3/servicehub-backend/internal/app"
	"go.uber.org/zap"
)

func main() {
	cfg := cmd.MustLoadConfig()
	zapLogger := cmd.MustLoadLogger(cfg.Log.Level)
	defer func(zapLogger *zap.Logger) {
		_ = zapLogger.Sync()
	}(zapLogger)
	if err := app.Run(cfg, zapLogger); err != nil {
		log.Fatalf("application error: %s", err)
	}
}
