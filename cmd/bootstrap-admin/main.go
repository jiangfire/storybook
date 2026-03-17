package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"git.neolidy.top/neo/storybook/internal/bootstrap"
	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
)

func main() {
	email := flag.String("email", "", "admin email, required")
	username := flag.String("username", "", "admin username, optional")
	password := flag.String("password", "", "admin password; required when creating a new admin")
	flag.Parse()

	if *email == "" {
		fmt.Fprintln(os.Stderr, "缺少必填参数: --email")
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}

	result, err := bootstrap.EnsureAdmin(db, bootstrap.EnsureAdminParams{
		Email:    *email,
		Username: *username,
		Password: *password,
	})
	if err != nil {
		log.Fatalf("bootstrap admin failed: %v", err)
	}

	action := "promoted"
	if result.Created {
		action = "created"
	}

	fmt.Printf(
		"admin %s: id=%d email=%s username=%s role_changed=%t password_changed=%t\n",
		action,
		result.UserID,
		result.Email,
		result.Username,
		result.RoleChanged,
		result.PasswordChanged,
	)
}
