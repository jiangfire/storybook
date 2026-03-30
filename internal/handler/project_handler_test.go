package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGetOverviewReturnsZeroWhenAveragePointsIsNull(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:project_handler_overview_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.ActivityLog{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{
		Username:       "owner",
		Email:          "owner@example.com",
		HashedPassword: "hashed-password",
		Role:           model.RoleProduct,
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}

	project := model.Project{
		Name:        "零点数项目",
		Description: "用于测试概览聚合空值",
		OwnerID:     owner.ID,
		AgileMode:   model.AgileModeKanban,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	h := NewProjectHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.GET("/api/projects/:id/overview", h.GetOverview)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/1/overview", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data missing: %#v", payload)
	}

	stats, ok := data["statistics"].(map[string]any)
	if !ok {
		t.Fatalf("statistics missing: %#v", data)
	}

	avgPoints, ok := stats["avg_story_points"].(float64)
	if !ok {
		t.Fatalf("avg_story_points is not float64: %#v", stats["avg_story_points"])
	}
	if avgPoints != 0 {
		t.Fatalf("expected avg_story_points=0, got %v", avgPoints)
	}
}

func TestParseAggregatedTime(t *testing.T) {
	t.Parallel()

	expected := time.Date(2026, 3, 23, 22, 10, 15, 0, time.UTC)
	testCases := []struct {
		name  string
		input any
		want  time.Time
		ok    bool
	}{
		{
			name:  "time value",
			input: expected,
			want:  expected,
			ok:    true,
		},
		{
			name:  "pointer time value",
			input: &expected,
			want:  expected,
			ok:    true,
		},
		{
			name:  "sqlite timestamp string",
			input: "2026-03-23 22:10:15",
			want:  expected,
			ok:    true,
		},
		{
			name:  "timestamp bytes",
			input: []byte("2026-03-23 22:10:15"),
			want:  expected,
			ok:    true,
		},
		{
			name:  "invalid value",
			input: 123,
			ok:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseAggregatedTime(tc.input)
			if ok != tc.ok {
				t.Fatalf("expected ok=%v, got %v", tc.ok, ok)
			}
			if !tc.ok {
				return
			}
			if !got.Equal(tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestQueryProjectMaxTimeHandlesSQLiteAggregateString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:project_handler_max_time_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserStory{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	updatedAt := time.Date(2026, 3, 23, 22, 10, 15, 0, time.UTC)
	story := model.UserStory{
		ProjectID:          2,
		Title:              "测试故事",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusBacklog,
		ReviewStatus:       model.ReviewStatusPending,
		CreatedBy:          1,
		AcceptanceCriteria: datatypes.JSON([]byte("[]")),
		Tags:               datatypes.JSON([]byte("[]")),
		CodeReferences:     datatypes.JSON([]byte("[]")),
		UpdatedAt:          updatedAt,
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	h := NewProjectHandler(db)
	got, ok := h.queryProjectMaxTime(&model.UserStory{}, "updated_at", 2)
	if !ok {
		t.Fatal("expected queryProjectMaxTime to return a value")
	}
	if !got.Equal(updatedAt) {
		t.Fatalf("expected %v, got %v", updatedAt, got)
	}
}

func TestGetProjectIncludesPendingStoriesInTotals(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:project_handler_pending_stats_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{
		Username:       "owner-pending",
		Email:          "owner-pending@example.com",
		HashedPassword: "hashed-password",
		Role:           model.RoleProduct,
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}

	project := model.Project{
		Name:        "待审批项目",
		Description: "用于测试 pending 统计",
		OwnerID:     owner.ID,
		AgileMode:   model.AgileModeKanban,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&model.ProjectMember{
		ProjectID:     project.ID,
		UserID:        owner.ID,
		RoleInProject: model.RoleProduct,
	}).Error; err != nil {
		t.Fatalf("create owner member: %v", err)
	}

	stories := []model.UserStory{
		{
			ProjectID:          project.ID,
			Title:              "待审批故事",
			StoryType:          model.StoryTypeFeature,
			Status:             model.StoryStatusPending,
			ReviewStatus:       model.ReviewStatusPending,
			CreatedBy:          owner.ID,
			AcceptanceCriteria: datatypes.JSON([]byte("[]")),
			Tags:               datatypes.JSON([]byte("[]")),
			CodeReferences:     datatypes.JSON([]byte("[]")),
		},
		{
			ProjectID:          project.ID,
			Title:              "已完成故事",
			StoryType:          model.StoryTypeFeature,
			Status:             model.StoryStatusDone,
			ReviewStatus:       model.ReviewStatusApproved,
			CreatedBy:          owner.ID,
			AcceptanceCriteria: datatypes.JSON([]byte("[]")),
			Tags:               datatypes.JSON([]byte("[]")),
			CodeReferences:     datatypes.JSON([]byte("[]")),
		},
	}
	if err := db.Create(&stories).Error; err != nil {
		t.Fatalf("create stories: %v", err)
	}

	h := NewProjectHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.GET("/api/projects/:id", h.GetProject)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data := payload["data"].(map[string]any)
	stats := data["statistics"].(map[string]any)
	statusBreakdown := stats["status_breakdown"].(map[string]any)

	if got := int(stats["total_stories"].(float64)); got != 2 {
		t.Fatalf("expected total_stories=2, got %d", got)
	}
	if got := int(statusBreakdown[model.StoryStatusPending].(float64)); got != 1 {
		t.Fatalf("expected pending count=1, got %d", got)
	}
}

func TestListMemberCandidatesOnlyAllowsOwnerAndExcludesExistingMembers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:project_handler_member_candidates_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{Username: "owner", Email: "owner+candidates@example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	member := model.User{Username: "member", Email: "member@example.com", HashedPassword: "hashed-password", Role: model.RoleDeveloper}
	candidate := model.User{Username: "candidate", Email: "candidate@example.com", HashedPassword: "hashed-password", Role: model.RoleTester}
	nonOwner := model.User{Username: "pm", Email: "pm@example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	if err := db.Create(&[]model.User{owner, member, candidate, nonOwner}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	var users []model.User
	if err := db.Order("id ASC").Find(&users).Error; err != nil {
		t.Fatalf("reload users: %v", err)
	}
	owner = users[0]
	member = users[1]
	candidate = users[2]
	nonOwner = users[3]

	project := model.Project{Name: "成员候选测试", Description: "test", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&[]model.ProjectMember{
		{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct},
		{ProjectID: project.ID, UserID: member.ID, RoleInProject: model.RoleDeveloper},
		{ProjectID: project.ID, UserID: nonOwner.ID, RoleInProject: model.RoleProduct},
	}).Error; err != nil {
		t.Fatalf("create project members: %v", err)
	}

	h := NewProjectHandler(db)

	ownerRouter := gin.New()
	ownerRouter.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	ownerRouter.GET("/api/projects/:id/member-candidates", h.ListMemberCandidates)

	ownerResp := httptest.NewRecorder()
	ownerReq := httptest.NewRequest(http.MethodGet, "/api/projects/1/member-candidates", nil)
	ownerRouter.ServeHTTP(ownerResp, ownerReq)

	if ownerResp.Code != http.StatusOK {
		t.Fatalf("expected owner request 200, got %d body=%s", ownerResp.Code, ownerResp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(ownerResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal owner response: %v", err)
	}
	data := payload["data"].(map[string]any)
	usersPayload := data["users"].([]any)
	if len(usersPayload) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(usersPayload))
	}
	if got := usersPayload[0].(map[string]any)["email"].(string); got != candidate.Email {
		t.Fatalf("expected candidate email %s, got %s", candidate.Email, got)
	}

	memberRouter := gin.New()
	memberRouter.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, nonOwner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	memberRouter.GET("/api/projects/:id/member-candidates", h.ListMemberCandidates)

	memberResp := httptest.NewRecorder()
	memberReq := httptest.NewRequest(http.MethodGet, "/api/projects/1/member-candidates", nil)
	memberRouter.ServeHTTP(memberResp, memberReq)

	if memberResp.Code != http.StatusForbidden {
		t.Fatalf("expected non-owner request 403, got %d body=%s", memberResp.Code, memberResp.Body.String())
	}
}
