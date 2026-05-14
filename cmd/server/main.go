package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/bootstrap"
	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/router"
)

const bootstrapAdminCommand = "bootstrap-admin"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == bootstrapAdminCommand {
		return runBootstrapAdmin(args[1:], stdout, stderr)
	}
	return runServer(stderr)
}

func runServer(stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "load config failed: %v\n", err)
		return 1
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	if err != nil {
		fmt.Fprintf(stderr, "create logger failed: %v\n", err)
		return 1
	}
	slog.SetDefault(logger)

	db, err := database.Connect(cfg)
	if err != nil {
		logger.Error("connect database failed", "driver", cfg.DBDriver, "error", err)
		return 1
	}

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	r := router.NewWithLogger(db, tokenManager, logger)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

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

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server exited", "error", err)
		return 1
	case <-quit:
		logger.Info("shutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		return 1
	}
	logger.Info("server exited")
	return 0
}

func runBootstrapAdmin(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(bootstrapAdminCommand, flag.ContinueOnError)
	fs.SetOutput(stderr)

	email := fs.String("email", "", "admin email, required")
	username := fs.String("username", "", "admin username, optional")
	password := fs.String("password", "", "admin password; required when creating a new admin")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *email == "" {
		fmt.Fprintln(stderr, "缺少必填参数: --email")
		fs.Usage()
		return 2
	}

	cfg, err := config.LoadForBootstrap()
	if err != nil {
		fmt.Fprintf(stderr, "load config failed: %v\n", err)
		return 1
	}

	db, err := database.Connect(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "connect database failed: %v\n", err)
		return 1
	}

	result, err := bootstrap.EnsureAdmin(db, bootstrap.EnsureAdminParams{
		Email:    *email,
		Username: *username,
		Password: *password,
	})
	if err != nil {
		fmt.Fprintf(stderr, "bootstrap admin failed: %v\n", err)
		return 1
	}

	action := "promoted"
	if result.Created {
		action = "created"
	}

	fmt.Fprintf(
		stdout,
		"admin %s: id=%d email=%s username=%s role_changed=%t password_changed=%t\n",
		action,
		result.UserID,
		result.Email,
		result.Username,
		result.RoleChanged,
		result.PasswordChanged,
	)
	return 0
}
