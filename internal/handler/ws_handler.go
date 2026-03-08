package handler

import (
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/realtime"
	"github.com/gin-gonic/gin"
)

type WSHandler struct {
	tokenManager *auth.TokenManager
	hub          *realtime.Hub
}

func NewWSHandler(tokenManager *auth.TokenManager, hub *realtime.Hub) *WSHandler {
	return &WSHandler{tokenManager: tokenManager, hub: hub}
}

func (h *WSHandler) Connect(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(header), "bearer ") {
			token = strings.TrimSpace(header[7:])
		}
	}

	if token == "" {
		api.Unauthorized(c, "未登录或Token无效")
		return
	}

	claims, err := h.tokenManager.ParseToken(token)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			api.TokenExpired(c, "Token过期")
		} else {
			api.Unauthorized(c, "未登录或Token无效")
		}
		return
	}

	if claims.Type != auth.TokenTypeAccess {
		api.Unauthorized(c, "未登录或Token无效")
		return
	}

	h.hub.HandleWS(c, claims.UserID)
}
