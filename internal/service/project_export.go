package service

import (
	"time"

	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/repository"
	"gorm.io/gorm"
)

// ProjectSnapshot is a point-in-time backup of a project and all related
// entities. It is intentionally plain so that callers can JSON-serialise it
// or transform it however they need.
type ProjectSnapshot struct {
	FormatVersion string                `json:"format_version"`
	ExportedAt    time.Time             `json:"exported_at"`
	Project       model.Project         `json:"project"`
	Members       []model.ProjectMember `json:"members"`
	Stories       []model.UserStory     `json:"stories"`
	Sprints       []model.Sprint        `json:"sprints"`
	Bugs          []model.BugReport     `json:"bugs"`
	Tasks         []model.Task          `json:"tasks"`
	TestCases     []model.TestCase      `json:"test_cases"`
}

// ExportSnapshot queries every entity belonging to projectID and assembles a
// complete snapshot. The returned snapshot includes archived stories so the
// backup is complete.
func ExportSnapshot(db *gorm.DB, projectID uint) (*ProjectSnapshot, error) {
	projectRepo := repository.NewProjectRepository(db)
	storyRepo := repository.NewStoryRepository(db)
	sprintRepo := repository.NewSprintRepository(db)
	bugRepo := repository.NewBugRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	testcaseRepo := repository.NewTestCaseRepository(db)

	project, err := projectRepo.FindByID(projectID)
	if err != nil {
		return nil, err
	}

	members, err := projectRepo.ListMembers(project.ID)
	if err != nil {
		return nil, err
	}

	var stories []model.UserStory
	if err := storyRepo.DB().Where("project_id = ?", project.ID).Find(&stories).Error; err != nil {
		return nil, err
	}

	storyIDs := make([]uint, 0, len(stories))
	for _, s := range stories {
		storyIDs = append(storyIDs, s.ID)
	}

	var sprints []model.Sprint
	if err := sprintRepo.DB().Where("project_id = ?", project.ID).Find(&sprints).Error; err != nil {
		return nil, err
	}

	var bugs []model.BugReport
	if err := bugRepo.DB().Where("project_id = ?", project.ID).Find(&bugs).Error; err != nil {
		return nil, err
	}

	var tasks []model.Task
	if err := taskRepo.DB().Where("project_id = ?", project.ID).Find(&tasks).Error; err != nil {
		return nil, err
	}

	testCases, err := testcaseRepo.ListByStoryIDs(storyIDs)
	if err != nil {
		return nil, err
	}

	return &ProjectSnapshot{
		FormatVersion: "1.0",
		ExportedAt:    time.Now(),
		Project:       *project,
		Members:       members,
		Stories:       stories,
		Sprints:       sprints,
		Bugs:          bugs,
		Tasks:         tasks,
		TestCases:     testCases,
	}, nil
}
