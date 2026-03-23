package main

import (
	"fmt"
	"log/slog"
	"os"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create logger failed: %v\n", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	db, err := database.Connect(cfg)
	if err != nil {
		logger.Error("connect database failed", "driver", cfg.DBDriver, "error", err)
		os.Exit(1)
	}

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	r := router.NewWithLogger(db, tokenManager, logger)

	logger.Info(
		"storybook backend is starting",
		"addr",
		cfg.ServerAddr,
		"db_driver",
		cfg.DBDriver,
		"log_level",
		cfg.LogLevel,
		"log_format",
		cfg.LogFormat,
	)
	if err := r.Run(cfg.ServerAddr); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}
