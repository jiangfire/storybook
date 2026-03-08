package main

import (
	"flag"
	"log"

	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/pkg/mcp"
)

func main() {
	configPath := flag.String("config", "", "mcp config file path")
	flag.Parse()

	runCfg, err := loadRunConfig(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	db, err := database.Connect(runCfg.DBConfig)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}

	switch runCfg.TransportType {
	case "stdio":
		server := mcp.NewServer(db)
		if err := server.Start(); err != nil {
			log.Fatalf("mcp stdio server exited: %v", err)
		}
	case "http":
		if err := startMCPHTTPServer(db, runCfg.HTTPAddr); err != nil {
			log.Fatalf("mcp http server exited: %v", err)
		}
	default:
		log.Fatalf("unsupported transport type: %s", runCfg.TransportType)
	}
}
