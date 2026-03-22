package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoginLockoutAfterFiveFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:auth_handler_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), 10)
	user := model.User{
		Username:       "tester",
		Email:          "tester@example.com",
		HashedPassword: string(hashed),
		Role:           model.RoleDeveloper,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	h := NewAuthHandler(db, auth.NewTokenManager("test-secret", 24, 24*7))

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.POST("/api/auth/login", h.Login)
		c.Request = newJSONRequest(t, http.MethodPost, "/api/auth/login", gin.H{
			"email":    user.Email,
			"password": "wrong1234",
		})
		r.ServeHTTP(w, c.Request)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 at attempt %d, got %d", i+1, w.Code)
		}
	}

	var refreshed model.User
	if err := db.First(&refreshed, user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if refreshed.LockedUntil == nil {
		t.Fatalf("expected user to be locked")
	}

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.POST("/api/auth/login", h.Login)
	c.Request = newJSONRequest(t, http.MethodPost, "/api/auth/login", gin.H{
		"email":    user.Email,
		"password": "Pass1234",
	})
	r.ServeHTTP(w, c.Request)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected locked user login to fail, got %d", w.Code)
	}
}

func TestRefreshRejectedAfterUserStateChanges(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:auth_refresh_state_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), 10)
	user := model.User{
		Username:       "refresh-user",
		Email:          "refresh@example.com",
		HashedPassword: string(hashed),
		Role:           model.RoleDeveloper,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	tokenManager := auth.NewTokenManager("test-secret", 24, 24*7)
	refreshToken, _, err := tokenManager.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	forcedUpdatedAt := time.Now().Add(2 * time.Second)
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"role":       model.RoleProduct,
		"updated_at": forcedUpdatedAt,
	}).Error; err != nil {
		t.Fatalf("update user: %v", err)
	}

	h := NewAuthHandler(db, tokenManager)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.POST("/api/auth/refresh", h.Refresh)
	c.Request = newJSONRequest(t, http.MethodPost, "/api/auth/refresh", gin.H{
		"refresh_token": refreshToken,
	})
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh to be rejected after user update, got %d body=%s", w.Code, w.Body.String())
	}
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	return req
}
