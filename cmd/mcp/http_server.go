package main

import (
	"net/http"

	"git.neolidy.top/neo/storybook/pkg/mcp"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func startMCPHTTPServer(db *gorm.DB, addr string) error {
	gin.EnableJsonDecoderDisallowUnknownFields()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	server := mcp.NewServer(db)
	r.Any("/mcp", gin.WrapH(http.HandlerFunc(server.ServeHTTP)))

	return r.Run(addr)
}
