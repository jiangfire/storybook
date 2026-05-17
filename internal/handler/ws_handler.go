package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/api"
	"github.com/jiangfire/storybook/internal/auth"
	"github.com/jiangfire/storybook/internal/realtime"
)

type WSHandler struct {
	tokenManager *auth.TokenManager
	hub          *realtime.Hub
}

func NewWSHandler(tokenManager *auth.TokenManager, hub *realtime.Hub) *WSHandler {
	return &WSHandler{tokenManager: tokenManager, hub: hub}
}

func (h *WSHandler) Connect(c *gin.Context) {
	token, protocol := parseWebSocketProtocolHeader(c.GetHeader("Sec-WebSocket-Protocol"))
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

	h.hub.HandleWS(c, claims.UserID, protocol)
}

func parseWebSocketProtocolHeader(header string) (token, protocol string) {
	parts := strings.Split(header, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			trimmed = append(trimmed, part)
		}
	}

	if len(trimmed) >= 2 && strings.EqualFold(trimmed[0], "storybook-token") {
		return trimmed[1], "storybook-token"
	}

	return "", ""
}
