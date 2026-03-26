package main

import (
	"flag"
	"log"

	"git.neolidy.top/neo/storybook/internal/database"
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

	if err := startMCPHTTPServer(db, runCfg.HTTPAddr); err != nil {
		log.Fatalf("mcp http server exited: %v", err)
	}
}
