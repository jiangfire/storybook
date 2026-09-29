package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

var (
	benchOnce   sync.Once
	benchRouter *gin.Engine
	benchErr    error
)

// BenchmarkSearchKeyword 建立关键字搜索端点的性能基线：
// 5 项目 / 500 故事 / 200 缺陷的中等规模数据集。
// 用法：go test ./internal/handler -bench BenchmarkSearch -benchmem
func BenchmarkSearchKeyword(b *testing.B) {
	gin.SetMode(gin.TestMode)

	// Go benchmark 框架会用递增的 b.N 多次调用本函数，种子只做一次
	benchOnce.Do(func() { benchRouter, benchErr = setupSearchBench() })
	if benchErr != nil {
		b.Fatal(benchErr)
	}
	router := benchRouter

	req := func() *http.Request {
		r, _ := http.NewRequest(http.MethodGet, "/api/search?q=登录&type=all&limit=50", nil)
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req())
		if w.Code != http.StatusOK {
			b.Fatalf("search failed: %d %s", w.Code, w.Body.String())
		}
	}
}

func setupSearchBench() (*gin.Engine, error) {
	db, err := gorm.Open(sqlite.Open("file:bench_search?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{},
		&model.UserStory{}, &model.BugReport{}); err != nil {
		return nil, err
	}

	admin := model.User{Username: "admin", Email: "admin@bench.dev", HashedPassword: "x", Role: model.RoleAdmin}
	if err := db.Create(&admin).Error; err != nil {
		return nil, err
	}
	for p := 1; p <= 5; p++ {
		if err := db.Create(&model.Project{Name: fmt.Sprintf("project-%d 登录重构", p), OwnerID: 1, AgileMode: model.AgileModeKanban}).Error; err != nil {
			return nil, err
		}
	}
	stories := make([]model.UserStory, 0, 500)
	for i := 1; i <= 500; i++ {
		stories = append(stories, model.UserStory{
			ProjectID: uint(i%5 + 1), Title: fmt.Sprintf("story-%d 支持手机号登录", i),
			StoryType: "feature", Status: model.StoryStatusBacklog, Priority: i%3 + 1,
			CreatedBy: 1, AcceptanceCriteria: []byte(`[]`), Tags: []byte(`[]`), CodeReferences: []byte(`[]`),
		})
	}
	if err := db.CreateInBatches(stories, 100).Error; err != nil {
		return nil, err
	}
	bugs := make([]model.BugReport, 0, 200)
	for i := 1; i <= 200; i++ {
		bugs = append(bugs, model.BugReport{
			ProjectID: uint(i%5 + 1), Title: fmt.Sprintf("bug-%d 登录失败", i),
			Severity: "low", Status: "open", ReportedBy: 1,
		})
	}
	if err := db.CreateInBatches(bugs, 100).Error; err != nil {
		return nil, err
	}

	handler := NewSearchHandler(db)
	router := gin.New()
	router.GET("/api/search", func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(1))
		c.Set(middleware.CtxRoleKey, model.RoleAdmin)
		c.Next()
	}, handler.Search)

	return router, nil
}
