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

	"github.com/jiangfire/storybook/internal/auth"
	"github.com/jiangfire/storybook/internal/bootstrap"
	"github.com/jiangfire/storybook/internal/config"
	"github.com/jiangfire/storybook/internal/database"
	"github.com/jiangfire/storybook/internal/logging"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/router"
	"github.com/jiangfire/storybook/internal/wiring"
)

const (
	cmdServer         = "server"
	cmdBootstrapAdmin = "bootstrap-admin"
)

var (
	binName = "storybook"
	version = "dev"
	commands = []struct{ name, short string }{
		{cmdServer, "启动 HTTP 服务器（默认子命令）"},
		{cmdBootstrapAdmin, "创建或提升管理员账户"},
	}
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runServer(stdout, stderr)
	}

	switch args[0] {
	case cmdServer:
		return runServerWithArgs(args[1:], stdout, stderr)
	case cmdBootstrapAdmin:
		return runBootstrapAdmin(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "%s %s\n", binName, version)
		return 0
	default:
		fmt.Fprintf(stderr, "未知子命令: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runServerWithArgs(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(cmdServer, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "用法: %s server\n\n", binName)
		fmt.Fprintf(stderr, "启动 storybook HTTP 服务器。\n\n")
		fmt.Fprintf(stderr, "配置通过环境变量或 .env 文件加载，无需额外 flags。\n")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	return runServer(stdout, stderr)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "%s - storybook 后端管理工具\n\n", binName)
	fmt.Fprintf(w, "用法:\n  %s <command> [flags]\n\n", binName)
	fmt.Fprintf(w, "子命令:\n")
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %-20s %s\n", cmd.name, cmd.short)
	}
	fmt.Fprintf(w, "\n全局标志:\n")
	fmt.Fprintf(w, "  %-20s 显示帮助信息\n", "-h, --help")
	fmt.Fprintf(w, "  %-20s 显示版本号\n", "-v, --version")
	fmt.Fprintf(w, "\n使用 \"%s <command> -h\" 查看子命令的详细用法。\n", binName)
}

func runServer(stdout, stderr io.Writer) int {
	config.LoadDotEnv()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "load config failed: %v\n", err)
		return 1
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat, stdout)
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
	container, err := wiring.Build(cfg, db, logger, tokenManager)
	if err != nil {
		logger.Error("wiring build failed", "error", err)
		return 1
	}
	r := router.New(container)

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
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
		defer close(errCh)
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
	config.LoadDotEnv()
	fs := flag.NewFlagSet(cmdBootstrapAdmin, flag.ContinueOnError)
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

	if err := db.AutoMigrate(&model.User{}); err != nil {
		fmt.Fprintf(stderr, "auto migrate failed: %v\n", err)
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
