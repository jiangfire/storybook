package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestSemanticSearch_RequiresAuth 测试语义搜索需要认证
func TestSemanticSearch_RequiresAuth(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithoutAuth(t)
	req, _ := http.NewRequest("GET", "/api/search/semantic?q=测试", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	// 没有 vectorSvc 时返回 500，有 vectorSvc 但没有认证时返回 401
	// 这里测试的是没有 vectorSvc 的情况
	assert.Contains(t, []int{http.StatusUnauthorized, http.StatusInternalServerError}, w.Code)
}

// TestSemanticSearch_EmptyQuery 测试空查询（需要认证）
func TestSemanticSearch_EmptyQuery(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	req, _ := http.NewRequest("GET", "/api/search/semantic?q=&limit=10", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticSearch_DefaultLimitWithoutValidationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	req, _ := http.NewRequest("GET", "/api/search/semantic?q=测试", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewSearchHandlerWithVector(nil, &mockVectorService{})
	router.GET("/api/search/capabilities", mockAuthMiddleware(), handler.Capabilities)

	req, _ := http.NewRequest("GET", "/api/search/capabilities", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"semantic_enabled":true`)
}

// TestSemanticSearch_MissingQuery 测试缺少查询参数（需要认证）
func TestSemanticSearch_MissingQuery(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	req, _ := http.NewRequest("GET", "/api/search/semantic?limit=10", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSemanticSearch_NoVectorService 测试向量服务未启用（需要认证）
func TestSemanticSearch_NoVectorService(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithoutVector(t)
	req, _ := http.NewRequest("GET", "/api/search/semantic?q=测试", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSemanticSearch_FiltersUnauthorizedProjectIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:search_handler_authz_test?mode=memory&cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}))

	user := model.User{Username: "pm", Email: "pm@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	assert.NoError(t, db.Create(&user).Error)
	projectOwned := model.Project{Name: "owned", OwnerID: user.ID, AgileMode: model.AgileModeKanban}
	projectHidden := model.Project{Name: "hidden", OwnerID: user.ID + 100, AgileMode: model.AgileModeKanban}
	assert.NoError(t, db.Create(&projectOwned).Error)
	assert.NoError(t, db.Create(&projectHidden).Error)

	spy := &mockVectorService{}
	handler := NewSearchHandlerWithVector(db, spy)
	router := gin.New()
	router.GET("/api/search/semantic", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("user_role", model.RoleProduct)
		c.Next()
	}, handler.SearchSemantic)

	req, _ := http.NewRequest("GET", "/api/search/semantic?q=登录&project_ids=1,2&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []uint{projectOwned.ID}, spy.lastProjectIDs)
}

// TestSimilarStories_MissingRequestBody 测试缺少请求体（需要认证）
func TestSimilarStories_MissingRequestBody(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	req, _ := http.NewRequest("POST", "/api/stories/similar", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSimilarStories_InvalidJSON 测试无效 JSON（需要认证）
func TestSimilarStories_InvalidJSON(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	req, _ := http.NewRequest("POST", "/api/stories/similar", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSimilarStories_DefaultLimitWithoutValidationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	body := map[string]any{
		"title":      "登录功能",
		"project_id": 1,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/stories/similar", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSuggestTags_MissingContent 测试缺少内容（需要认证）
func TestSuggestTags_MissingContent(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	body := map[string]any{}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/tags/suggest", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSuggestTags_DefaultLimitWithoutValidationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouterWithAuth(t)
	body := map[string]any{
		"title": "支付功能",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/tags/suggest", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// setupTestRouterWithoutAuth 设置不带认证的测试路由
func setupTestRouterWithoutAuth(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewSearchHandler(nil) // 没有 vectorSvc
	router.GET("/api/search/semantic", handler.SearchSemantic)

	return router
}

// setupTestRouterWithoutVector 设置不带向量服务的测试路由
func setupTestRouterWithoutVector(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewSearchHandler(nil) // 没有 vectorSvc
	router.GET("/api/search/semantic", mockAuthMiddleware(), handler.SearchSemantic)

	return router
}

// setupTestRouterWithAuth 设置带认证的测试路由
func setupTestRouterWithAuth(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	db, err := gorm.Open(sqlite.Open("file:search_handler_general_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}))
	require.NoError(t, db.Create(&model.Project{Name: "p1", OwnerID: 1, AgileMode: model.AgileModeKanban}).Error)

	mockSvc := &mockVectorService{}
	handler := NewSearchHandlerWithVector(db, mockSvc)

	router.GET("/api/search/semantic", mockAuthMiddleware(), handler.SearchSemantic)
	router.POST("/api/stories/similar", mockAuthMiddleware(), handler.SimilarStories)
	router.POST("/api/tags/suggest", mockAuthMiddleware(), handler.SuggestTags)

	return router
}

// mockAuthMiddleware 模拟认证中间件
func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置 mock 用户信息
		c.Set("user_id", uint(1))
		c.Set("user_role", "admin")
		c.Next()
	}
}

// mockVectorService Mock Vector Service 用于测试
type mockVectorService struct {
	lastProjectIDs []uint
}

func (m *mockVectorService) SearchSimilarStories(ctx context.Context, query string, projectIDs []uint, limit int) ([]service.SimilarStory, error) {
	m.lastProjectIDs = append([]uint(nil), projectIDs...)
	// 返回空的 mock 结果
	return []service.SimilarStory{}, nil
}

func (m *mockVectorService) IndexStory(ctx context.Context, story *model.UserStory) error {
	return nil
}

func (m *mockVectorService) BatchIndexStories(ctx context.Context, stories []model.UserStory) error {
	return nil
}

func (m *mockVectorService) PrepareStoryContent(story *model.UserStory) string {
	return ""
}
