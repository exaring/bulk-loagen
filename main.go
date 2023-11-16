package main

import (
	_ "embed"
	"net/http"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/exaring/bulk-loagen/pkg/bulkloagen"
	"github.com/exaring/bulk-loagen/pkg/config"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	cfg := &config.Config{}

	parser, err := kong.New(cfg, kong.Configuration(kongyaml.Loader, "/config.yaml", "config.yaml"))
	if err != nil {
		logger.Fatal("loading config.yaml", zap.Error(err))
	}

	_, err = parser.Parse(nil)
	if err != nil {
		logger.Fatal("parsing config", zap.Error(err))
	}

	svc, err := bulkloagen.NewService(cfg)
	if err != nil {
		logger.Fatal("instantiating service", zap.Error(err))
	}

	if err := http.ListenAndServe(cfg.ListenAddr, svc); err != nil {
		logger.Fatal("listening", zap.String("listenAddr", cfg.ListenAddr), zap.Error(err))
	}
}
