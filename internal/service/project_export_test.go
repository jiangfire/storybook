package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestExportSnapshotAssemblesAllEntities(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:export_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.UserStory{},
		&model.Sprint{},
		&model.BugReport{},
		&model.Task{},
		&model.TestCase{},
	))

	owner := model.User{Email: "owner@example.com", Role: model.RoleProduct}
	require.NoError(t, db.Create(&owner).Error)

	project := model.Project{Name: "Export Me", OwnerID: owner.ID, AgileMode: model.AgileModeScrum}
	require.NoError(t, db.Create(&project).Error)

	member := model.ProjectMember{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct}
	require.NoError(t, db.Create(&member).Error)

	story := model.UserStory{
		ProjectID:          project.ID,
		Title:              "Story 1",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusDone,
		ReviewStatus:       model.ReviewStatusApproved,
		CreatedBy:          owner.ID,
		AcceptanceCriteria: datatypes.JSON([]byte("[]")),
		Tags:               datatypes.JSON([]byte("[]")),
		CodeReferences:     datatypes.JSON([]byte("[]")),
	}
	require.NoError(t, db.Create(&story).Error)

	sprint := model.Sprint{ProjectID: project.ID, Name: "S1", Status: model.SprintStatusActive, CreatedBy: owner.ID}
	require.NoError(t, db.Create(&sprint).Error)

	bug := model.BugReport{ProjectID: project.ID, Title: "Bug 1", Severity: model.BugSeverityHigh, ReportedBy: owner.ID}
	require.NoError(t, db.Create(&bug).Error)

	task := model.Task{ProjectID: project.ID, StoryID: story.ID, Title: "Task 1", Status: model.TaskStatusDone, CreatedBy: owner.ID, CodeReferences: datatypes.JSON([]byte("[]"))}
	require.NoError(t, db.Create(&task).Error)

	tc := model.TestCase{StoryID: story.ID, Title: "TC 1", Steps: datatypes.JSON([]byte("[]")), CreatedBy: owner.ID}
	require.NoError(t, db.Create(&tc).Error)

	snapshot, err := ExportSnapshot(db, project.ID)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	require.Equal(t, "1.0", snapshot.FormatVersion)
	require.False(t, snapshot.ExportedAt.IsZero())
	require.Equal(t, project.ID, snapshot.Project.ID)

	require.Len(t, snapshot.Members, 1)
	require.Equal(t, owner.ID, snapshot.Members[0].UserID)

	require.Len(t, snapshot.Stories, 1)
	require.Equal(t, story.ID, snapshot.Stories[0].ID)

	require.Len(t, snapshot.Sprints, 1)
	require.Equal(t, sprint.ID, snapshot.Sprints[0].ID)

	require.Len(t, snapshot.Bugs, 1)
	require.Equal(t, bug.ID, snapshot.Bugs[0].ID)

	require.Len(t, snapshot.Tasks, 1)
	require.Equal(t, task.ID, snapshot.Tasks[0].ID)

	require.Len(t, snapshot.TestCases, 1)
	require.Equal(t, tc.ID, snapshot.TestCases[0].ID)
}

func TestExportSnapshotReturnsErrorForMissingProject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:export_missing_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.UserStory{},
		&model.Sprint{},
		&model.BugReport{},
		&model.Task{},
		&model.TestCase{},
	))

	_, err = ExportSnapshot(db, 9999)
	require.Error(t, err)
}
