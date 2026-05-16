package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeleteStoryByCreator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:story_delete_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.ActivityLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	creator := model.User{Username: "pm", Email: "pm2@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&creator).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := model.Project{Name: "p2", OwnerID: creator.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	story := model.UserStory{
		ProjectID:          project.ID,
		Title:              "story",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusBacklog,
		Priority:           1,
		CreatedBy:          creator.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{}),
		Tags:               model.MarshalJSON([]string{}),
		CodeReferences:     model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	h := NewStoryHandler(db, nil)
	r := gin.New()
	r.DELETE("/api/stories/:id", func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, creator.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Set(middleware.CtxStoryKey, &story)
		c.Set(middleware.CtxProjectKey, &project)
		c.Set(middleware.CtxIsOwnerKey, project.OwnerID == creator.ID)
		h.DeleteStory(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/stories/%d", story.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var count int64
	if err := db.Model(&model.UserStory{}).Where("id = ?", story.ID).Count(&count).Error; err != nil {
		t.Fatalf("count story: %v", err)
	}
	if count != 0 {
		t.Fatalf("story not deleted")
	}
}
