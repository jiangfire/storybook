package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	maxLoginAttempts = 5
	lockDuration     = 15 * time.Minute
)

type AuthHandler struct {
	db           *gorm.DB
	tokenManager *auth.TokenManager
}

func NewAuthHandler(db *gorm.DB, tokenManager *auth.TokenManager) *AuthHandler {
	return &AuthHandler{db: db, tokenManager: tokenManager}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=product developer tester"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if !isStrongPassword(req.Password) {
		api.BadRequest(c, "参数验证失败", api.ErrorItem{Field: "password", Message: "密码至少8位，包含字母和数字"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var exists int64
	if err := h.db.Model(&model.User{}).Where("email = ?", email).Count(&exists).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if exists > 0 {
		api.Conflict(c, "邮箱已被注册")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	username := strings.Split(email, "@")[0]
	if username == "" {
		username = fmt.Sprintf("user_%d", time.Now().Unix())
	}

	user := model.User{
		Username:       fmt.Sprintf("%s_%d", username, time.Now().UnixNano()%100000),
		Email:          email,
		HashedPassword: string(hashed),
		Role:           req.Role,
	}

	if err := h.db.Create(&user).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	accessToken, accessExp, err := h.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	refreshToken, _, err := h.tokenManager.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "注册成功", gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
		"token":         accessToken,
		"refresh_token": refreshToken,
		"expires_at":    accessExp,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user model.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.Unauthorized(c, "邮箱或密码错误")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	now := time.Now()
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		api.Unauthorized(c, "账号已锁定，请稍后重试")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		updates := map[string]any{
			"failed_login_attempts": user.FailedLoginAttempts + 1,
		}

		if user.FailedLoginAttempts+1 >= maxLoginAttempts {
			updates["locked_until"] = now.Add(lockDuration)
		}

		_ = h.db.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error
		api.Unauthorized(c, "邮箱或密码错误")
		return
	}

	resetUpdates := map[string]any{
		"failed_login_attempts": 0,
		"locked_until":          nil,
		"last_login_at":         now,
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", user.ID).Updates(resetUpdates).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	accessToken, accessExp, err := h.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	refreshToken, _, err := h.tokenManager.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "登录成功", gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
		"token":         accessToken,
		"refresh_token": refreshToken,
		"expires_at":    accessExp,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	claims, err := h.tokenManager.ParseToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			api.TokenExpired(c, "Token过期")
		} else {
			api.Unauthorized(c, "Refresh Token无效")
		}
		return
	}
	if claims.Type != auth.TokenTypeRefresh {
		api.Unauthorized(c, "Refresh Token无效")
		return
	}

	accessToken, accessExp, err := h.tokenManager.GenerateAccessToken(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "Token刷新成功", gin.H{
		"token":      accessToken,
		"expires_at": accessExp,
	})
}

func isStrongPassword(p string) bool {
	if len(p) < 8 {
		return false
	}

	hasLetter := false
	hasDigit := false
	for _, r := range p {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
