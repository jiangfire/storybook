package mcp

import (
	"context"
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestServerGetStory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mcp_pkg_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "u", Email: "u@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	project := model.Project{Name: "p", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	story := model.UserStory{
		ProjectID: project.ID,
		Title:     "story",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-1", Description: "desc", Status: model.ACStatusPending, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	s := NewServer(db)
	resp := s.handleRequest(context.Background(), Request{
		ID:   "1",
		Tool: "get_story",
		Arguments: map[string]any{
			"story_id": float64(story.ID),
		},
	})
	if !resp.Success {
		t.Fatalf("expect success, got error: %s", resp.Error)
	}
}

func TestServerValidateACRejectInvalidStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mcp_pkg_invalid_status_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "u2", Email: "u2@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	project := model.Project{Name: "p2", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	story := model.UserStory{
		ProjectID: project.ID,
		Title:     "story",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-1", Description: "desc", Status: model.ACStatusPending, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	s := NewServer(db)
	resp := s.handleRequest(context.Background(), Request{
		ID:   "2",
		Tool: "validate_ac",
		Arguments: map[string]any{
			"story_id": float64(story.ID),
			"ac_id":    "ac-1",
			"status":   "invalid_status",
		},
	})
	if resp.Success {
		t.Fatalf("expected failure for invalid status")
	}
}

func TestServerAnalyzeCodeACRejectPathTraversal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mcp_pkg_traversal_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "u3", Email: "u3@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	project := model.Project{Name: "p3", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	story := model.UserStory{
		ProjectID: project.ID,
		Title:     "story",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-1", Description: "desc", Status: model.ACStatusPending, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	s := NewServer(db)
	resp := s.handleRequest(context.Background(), Request{
		ID:   "3",
		Tool: "analyze_code_ac",
		Arguments: map[string]any{
			"story_id":  float64(story.ID),
			"file_path": "..\\go.mod",
		},
	})
	if resp.Success {
		t.Fatalf("expected failure for path traversal")
	}
}
