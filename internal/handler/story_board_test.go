package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGetBoardIncludesPendingColumn(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:story_board_pending_column_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.BoardColumn{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{Username: "owner", Email: "owner@story-board.example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}

	project := model.Project{Name: "看板待审批", Description: "test", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&model.ProjectMember{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct}).Error; err != nil {
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
			Title:              "待办故事",
			StoryType:          model.StoryTypeFeature,
			Status:             model.StoryStatusBacklog,
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

	h := NewStoryHandler(db, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.GET("/api/projects/:id/board", h.GetBoard)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/1/board", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data := payload["data"].(map[string]any)
	columns := data["columns"].([]any)
	if len(columns) == 0 {
		t.Fatal("expected board columns")
	}

	first := columns[0].(map[string]any)
	if got := first["status"].(string); got != model.StoryStatusPending {
		t.Fatalf("expected first column to be pending, got %s", got)
	}
	if got := int(first["count"].(float64)); got != 1 {
		t.Fatalf("expected pending count=1, got %d", got)
	}
}
