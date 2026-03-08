package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// distFS 内嵌前端构建产物。
//
//go:embed dist
var embeddedDist embed.FS

// Register 在 Gin 引擎上注册前端静态资源与 SPA 回退路由。
// 除 /api、/mcp、/ws 外，未命中的 GET/HEAD 请求都会尝试返回前端资源。
func Register(r *gin.Engine) {
	distFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		panic("load embedded frontend dist failed: " + err.Error())
	}

	indexFile := resolveIndexFile(distFS)
	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if isBackendPath(requestPath) {
			c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
		if cleanPath == "" || cleanPath == "." {
			serveEmbeddedFile(c, fileServer, indexFile)
			return
		}

		if _, statErr := fs.Stat(distFS, cleanPath); statErr == nil {
			serveEmbeddedFile(c, fileServer, cleanPath)
			return
		}

		// 带扩展名但不存在的资源按 404 处理，避免把丢失的 JS/CSS 回退到 index.html。
		if path.Ext(cleanPath) != "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		serveEmbeddedFile(c, fileServer, indexFile)
	})
}

func serveEmbeddedFile(c *gin.Context, fileServer http.Handler, filePath string) {
	originalPath := c.Request.URL.Path
	if filePath == "index.html" || filePath == "index.generated.html" {
		c.Request.URL.Path = "/"
	} else {
		c.Request.URL.Path = "/" + filePath
	}
	fileServer.ServeHTTP(c.Writer, c.Request)
	c.Request.URL.Path = originalPath
}

func resolveIndexFile(distFS fs.FS) string {
	if _, err := fs.Stat(distFS, "index.generated.html"); err == nil {
		return "index.generated.html"
	}
	return "index.html"
}

func isBackendPath(requestPath string) bool {
	return requestPath == "/api" ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/mcp" ||
		strings.HasPrefix(requestPath, "/mcp/") ||
		requestPath == "/ws" ||
		strings.HasPrefix(requestPath, "/ws/")
}
