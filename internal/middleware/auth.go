package middleware

import (
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	CtxUserIDKey = "user_id"
	CtxEmailKey  = "email"
	CtxRoleKey   = "role"
)

func AuthRequired(tokenManager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			api.Unauthorized(c, "未登录或Token无效")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			api.Unauthorized(c, "未登录或Token无效")
			c.Abort()
			return
		}

		claims, err := tokenManager.ParseToken(parts[1])
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				api.TokenExpired(c, "Token过期")
			} else {
				api.Unauthorized(c, "未登录或Token无效")
			}
			c.Abort()
			return
		}
		if claims.Type != auth.TokenTypeAccess {
			api.Unauthorized(c, "未登录或Token无效")
			c.Abort()
			return
		}

		c.Set(CtxUserIDKey, claims.UserID)
		c.Set(CtxEmailKey, claims.Email)
		c.Set(CtxRoleKey, claims.Role)
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		rawRole, exists := c.Get(CtxRoleKey)
		if !exists {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		role, ok := rawRole.(string)
		if !ok {
			api.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		if _, ok := allowed[role]; !ok {
			api.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (uint, bool) {
	raw, ok := c.Get(CtxUserIDKey)
	if !ok {
		return 0, false
	}

	userID, ok := raw.(uint)
	return userID, ok
}

func CurrentRole(c *gin.Context) (string, bool) {
	raw, ok := c.Get(CtxRoleKey)
	if !ok {
		return "", false
	}

	role, ok := raw.(string)
	return role, ok
}
