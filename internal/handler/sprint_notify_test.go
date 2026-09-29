package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type notifyCall struct {
	userIDs []uint
	ev      service.NotificationEvent
}

type mockNotifier struct {
	notifyOne  []notifyCall
	notifyMany []notifyCall
}

func (m *mockNotifier) Notify(_ context.Context, userID uint, ev service.NotificationEvent) {
	m.notifyOne = append(m.notifyOne, notifyCall{userIDs: []uint{userID}, ev: ev})
}
func (m *mockNotifier) NotifyMany(_ context.Context, userIDs []uint, ev service.NotificationEvent) {
	m.notifyMany = append(m.notifyMany, notifyCall{userIDs: userIDs, ev: ev})
}
func (m *mockNotifier) NotifyProjectMembers(_ context.Context, _ uint, _ service.NotificationEvent) {}

// setupSprintNotifyEnv 建一个 sprint + 一个已指派给 developer 的故事（已在该冲刺内）。
func setupSprintNotifyEnv(t *testing.T) (*gorm.DB, *mockNotifier, *gin.Engine, model.Sprint, uint, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.Sprint{},
	))

	pm := model.User{Username: "pm", Email: "pm@test.dev", HashedPassword: "x", Role: model.RoleProduct}
	dev := model.User{Username: "dev", Email: "dev@test.dev", HashedPassword: "x", Role: model.RoleDeveloper}
	require.NoError(t, db.Create(&pm).Error)
	require.NoError(t, db.Create(&dev).Error)

	project := model.Project{Name: "p", OwnerID: pm.ID, AgileMode: model.AgileModeKanban}
	require.NoError(t, db.Create(&project).Error)

	sprint := model.Sprint{ProjectID: project.ID, Name: "sprint-1", Goal: "g",
		StartDate: time.Now(), EndDate: time.Now().Add(72 * time.Hour), Status: model.SprintStatusPlanned}
	require.NoError(t, db.Create(&sprint).Error)

	story := model.UserStory{ProjectID: project.ID, Title: "story-1", StoryType: "feature",
		Status: model.StoryStatusBacklog, Priority: 1, CreatedBy: pm.ID,
		AssignedTo: &dev.ID, AcceptanceCriteria: []byte(`[]`), Tags: []byte(`[]`), CodeReferences: []byte(`[]`),
		SprintID: &sprint.ID}
	require.NoError(t, db.Create(&story).Error)

	notifier := &mockNotifier{}
	h := NewSprintHandler(db).WithNotifier(notifier)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, pm.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.PATCH("/api/stories/:id/sprint", func(c *gin.Context) {
		var story model.UserStory
		require.NoError(t, db.First(&story, c.Param("id")).Error)
		c.Set(middleware.CtxStoryKey, &story)
		var project model.Project
		require.NoError(t, db.First(&project, story.ProjectID).Error)
		c.Set(middleware.CtxProjectKey, &project)
		c.Next()
	}, h.AssignStory)
	r.DELETE("/api/sprints/:id", func(c *gin.Context) {
		var s model.Sprint
		require.NoError(t, db.First(&s, c.Param("id")).Error)
		c.Set(middleware.CtxSprintKey, &s)
		c.Next()
	}, h.Delete)

	return db, notifier, r, sprint, pm.ID, dev.ID
}

func TestAssignStoryNotifiesStoryAssignee(t *testing.T) {
	db, notifier, r, sprint, _, devID := setupSprintNotifyEnv(t)

	// 规划进冲刺：应通知故事负责人
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/stories/1/sprint",
		bytes.NewBufferString(`{"sprint_id":`+jsonUint(sprint.ID)+`}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, notifier.notifyOne, 1)
	assert.Equal(t, devID, notifier.notifyOne[0].userIDs[0])
	assert.Equal(t, model.NotificationEntitySprint, notifier.notifyOne[0].ev.EntityType)
	assert.Contains(t, notifier.notifyOne[0].ev.Title, "sprint-1")

	// 移出冲刺：也应通知
	notifier.notifyOne = nil
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPatch, "/api/stories/1/sprint",
		bytes.NewBufferString(`{"sprint_id":null}`)))
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	require.Len(t, notifier.notifyOne, 1)
	assert.Equal(t, devID, notifier.notifyOne[0].userIDs[0])

	var _ = db
}

func TestDeleteSprintNotifiesAffectedAssignees(t *testing.T) {
	db, notifier, r, sprint, pmID, devID := setupSprintNotifyEnv(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/sprints/"+jsonUint(sprint.ID), nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, notifier.notifyMany, 1)
	assert.Contains(t, notifier.notifyMany[0].userIDs, devID)
	// 操作者（PM）不应收到自己的通知
	for _, uid := range notifier.notifyMany[0].userIDs {
		assert.NotEqual(t, pmID, uid)
	}

	var stories int64
	require.NoError(t, db.Model(&model.UserStory{}).Where("sprint_id = ?", sprint.ID).Count(&stories).Error)
	assert.Zero(t, stories)
}

func jsonUint(v uint) string {
	b, _ := json.Marshal(v)
	return string(b)
}
