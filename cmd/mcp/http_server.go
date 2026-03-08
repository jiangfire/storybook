package main

import (
	"git.neolidy.top/neo/storybook/internal/handler"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func startMCPHTTPServer(db *gorm.DB, addr string) error {
	gin.EnableJsonDecoderDisallowUnknownFields()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	mcpHandler := handler.NewMCPHandler(db)
	mcp := r.Group("/mcp")
	{
		mcp.GET("/health", mcpHandler.Health)
		mcp.GET("/stories/:id/acceptance-criteria", mcpHandler.GetStoryAC)
		mcp.GET("/stories/:id/ac-coverage", mcpHandler.ACCoverage)
		mcp.POST("/stories/:id/acceptance-criteria/:acID/status", mcpHandler.Validate)
		mcp.POST("/v1/stories/:id/validate", mcpHandler.Validate)
		mcp.GET("/v1/stories/:id", mcpHandler.GetStory)
		mcp.GET("/v1/projects/:id/stories", mcpHandler.ListStories)
		mcp.POST("/v1/stories/:id/update-ac-status", mcpHandler.BatchUpdateACStatus)
		mcp.POST("/v1/stories/:id/generate-ac-tests", mcpHandler.GenerateACTests)
		mcp.POST("/v1/stories/:id/analyze-code-ac", mcpHandler.AnalyzeCodeAC)
		mcp.GET("/v1/stats/ac-completion", mcpHandler.ACCompletionStats)
	}

	return r.Run(addr)
}
