package main

import (
	"fmt"
	"os"
	"strings"

	"git.neolidy.top/neo/storybook/internal/config"
	"gopkg.in/yaml.v3"
)

type runConfig struct {
	TransportType string
	HTTPAddr      string
	DBConfig      *config.Config
}

type mcpFileConfig struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Transport struct {
		Type string `yaml:"type"`
		Addr string `yaml:"addr"`
	} `yaml:"transport"`
	Database struct {
		Driver   string `yaml:"driver"`
		DSN      string `yaml:"dsn"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Name     string `yaml:"name"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`
}

func loadRunConfig(configPath string) (*runConfig, error) {
	base, err := config.Load()
	if err != nil {
		return nil, err
	}

	rc := &runConfig{
		TransportType: "stdio",
		HTTPAddr:      ":8081",
		DBConfig:      base,
	}

	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return rc, nil
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var fileCfg mcpFileConfig
	if err := yaml.Unmarshal(raw, &fileCfg); err != nil {
		return nil, err
	}

	if t := strings.ToLower(strings.TrimSpace(fileCfg.Transport.Type)); t != "" {
		rc.TransportType = t
	}
	if addr := strings.TrimSpace(fileCfg.Transport.Addr); addr != "" {
		rc.HTTPAddr = addr
	} else if addr := strings.TrimSpace(fileCfg.Server.Addr); addr != "" {
		rc.HTTPAddr = addr
	}

	if driver := strings.TrimSpace(fileCfg.Database.Driver); driver != "" {
		rc.DBConfig.DBDriver = driver
	}

	if dsn := strings.TrimSpace(fileCfg.Database.DSN); dsn != "" {
		rc.DBConfig.DBDSN = dsn
	} else if host := strings.TrimSpace(fileCfg.Database.Host); host != "" {
		port := fileCfg.Database.Port
		if port <= 0 {
			port = 5432
		}
		dbName := strings.TrimSpace(fileCfg.Database.Name)
		if dbName == "" {
			dbName = "storybook"
		}
		user := strings.TrimSpace(fileCfg.Database.User)
		if user == "" {
			user = "postgres"
		}
		sslmode := strings.TrimSpace(fileCfg.Database.SSLMode)
		if sslmode == "" {
			sslmode = "disable"
		}

		rc.DBConfig.DBDriver = "postgres"
		rc.DBConfig.DBDSN = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, fileCfg.Database.Password, dbName, sslmode,
		)
	}

	return rc, nil
}
