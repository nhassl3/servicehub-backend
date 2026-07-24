package main

import (
	"log"

	"github.com/nhassl3/servicehub-backend/internal/app"
	"go.uber.org/zap"
)

func main() {
	cfg := MustLoadConfig()
	zapLogger := MustLoadLogger(cfg.Log.Level)
	defer func(log *zap.Logger) {
		err := log.Sync()
		if err != nil {
			log.Fatal("logger sync error", zap.Error(err))
		}
	}(zapLogger)
	if err := app.Run(cfg, zapLogger); err != nil {
		log.Fatalf("application error: %s", err)
	}
}
