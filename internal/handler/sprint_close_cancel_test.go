package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// setupSprintTerminationTest builds an in-memory DB with one project, one
// sprint (given status) and stories in the given statuses all attached to
// that sprint. Returns the db, the sprint id, the created stories and a
// ready-to-use router.
func setupSprintTerminationTest(t *testing.T, dbName, sprintStatus string, storyStatuses []string) (*gorm.DB, uint, []model.UserStory, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.Sprint{}, &model.ActivityLog{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{Username: "pm", Email: "pm@" + dbName + ".example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}

	project := model.Project{Name: "冲刺收尾测试", Description: "test", OwnerID: owner.ID, AgileMode: model.AgileModeScrum}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&model.ProjectMember{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct}).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	now := time.Now().UTC()
	sprint := model.Sprint{
		ProjectID: project.ID,
		Name:      "冲刺一",
		StartDate: now.AddDate(0, 0, -14),
		EndDate:   now,
		Status:    sprintStatus,
		CreatedBy: owner.ID,
	}
	if err := db.Create(&sprint).Error; err != nil {
		t.Fatalf("create sprint: %v", err)
	}

	stories := make([]model.UserStory, 0, len(storyStatuses))
	for i, status := range storyStatuses {
		story := model.UserStory{
			ProjectID:          project.ID,
			Title:              "故事" + string(rune('A'+i)),
			StoryType:          model.StoryTypeFeature,
			Status:             status,
			ReviewStatus:       model.ReviewStatusApproved,
			CreatedBy:          owner.ID,
			SprintID:           &sprint.ID,
			AcceptanceCriteria: datatypes.JSON([]byte("[]")),
			Tags:               datatypes.JSON([]byte("[]")),
			CodeReferences:     datatypes.JSON([]byte("[]")),
		}
		if err := db.Create(&story).Error; err != nil {
			t.Fatalf("create story %d: %v", i, err)
		}
		stories = append(stories, story)
	}

	h := NewSprintHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Set(middleware.CtxProjectKey, &project)
		c.Set(middleware.CtxSprintKey, &sprint)
		c.Next()
	})
	r.POST("/sprints/:id/close", h.Close)
	r.POST("/sprints/:id/cancel", h.Cancel)

	return db, sprint.ID, stories, r
}

func mustReloadStory(t *testing.T, db *gorm.DB, id uint) model.UserStory {
	t.Helper()
	var story model.UserStory
	if err := db.First(&story, id).Error; err != nil {
		t.Fatalf("reload story %d: %v", id, err)
	}
	return story
}

func TestCloseSprintRevertsUndoneStories(t *testing.T) {
	db, sprintID, stories, r := setupSprintTerminationTest(t, "sprint_close_test", model.SprintStatusActive, []string{
		model.StoryStatusDone,
		model.StoryStatusInProgress,
		model.StoryStatusReady,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sprints/1/close", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// done 的故事保留冲刺归属，velocity 报表才能正确计入已完成点数。
	doneStory := mustReloadStory(t, db, stories[0].ID)
	if doneStory.SprintID == nil || *doneStory.SprintID != sprintID {
		t.Fatalf("done story should keep sprint attachment, got %+v", doneStory.SprintID)
	}

	// 未完成的故事脱离冲刺（sprint_id = NULL），状态本身不重置。
	for _, idx := range []int{1, 2} {
		story := mustReloadStory(t, db, stories[idx].ID)
		if story.SprintID != nil {
			t.Fatalf("undone story %q should be detached from sprint, got sprint_id=%d", story.Title, *story.SprintID)
		}
		if story.Status != stories[idx].Status {
			t.Fatalf("undone story %q status should be preserved as %q, got %q", story.Title, stories[idx].Status, story.Status)
		}
	}

	var sprint model.Sprint
	if err := db.First(&sprint, sprintID).Error; err != nil {
		t.Fatalf("reload sprint: %v", err)
	}
	if sprint.Status != model.SprintStatusCompleted {
		t.Fatalf("sprint should be completed, got %q", sprint.Status)
	}
}

func TestCancelSprintDetachesAllStories(t *testing.T) {
	db, sprintID, stories, r := setupSprintTerminationTest(t, "sprint_cancel_test", model.SprintStatusActive, []string{
		model.StoryStatusDone,
		model.StoryStatusInProgress,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sprints/1/cancel", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	for _, s := range stories {
		story := mustReloadStory(t, db, s.ID)
		if story.SprintID != nil {
			t.Fatalf("story %q should be detached after cancel, got sprint_id=%d", story.Title, *story.SprintID)
		}
	}

	var sprint model.Sprint
	if err := db.First(&sprint, sprintID).Error; err != nil {
		t.Fatalf("reload sprint: %v", err)
	}
	if sprint.Status != model.SprintStatusCancelled {
		t.Fatalf("sprint should be cancelled, got %q", sprint.Status)
	}
}

func TestCloseSprintRejectsNonActive(t *testing.T) {
	_, _, _, r := setupSprintTerminationTest(t, "sprint_close_reject_test", model.SprintStatusPlanned, []string{model.StoryStatusBacklog})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sprints/1/close", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 closing a planned sprint, got %d body=%s", w.Code, w.Body.String())
	}
}
