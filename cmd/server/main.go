package main

import (
	"log"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	r := router.New(db, tokenManager)

	log.Printf("storybook backend is running on %s", cfg.ServerAddr)
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
