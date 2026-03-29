package service

import (
	"errors"
	"fmt"
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type accessFixture struct {
	db       *gorm.DB
	owner    model.User
	admin    model.User
	member   model.User
	techLead model.User
	outsider model.User
	project1 model.Project
	project2 model.Project
	project3 model.Project
	story    model.UserStory
}

func newAccessFixture(t *testing.T) accessFixture {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectTechLead{},
		&model.UserStory{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	users := []model.User{
		{Username: "owner", Email: "owner@example.com", HashedPassword: "hashed", Role: model.RoleProduct},
		{Username: "admin", Email: "admin@example.com", HashedPassword: "hashed", Role: model.RoleAdmin},
		{Username: "member", Email: "member@example.com", HashedPassword: "hashed", Role: model.RoleDeveloper},
		{Username: "tech", Email: "tech@example.com", HashedPassword: "hashed", Role: model.RoleTechLead},
		{Username: "outsider", Email: "outsider@example.com", HashedPassword: "hashed", Role: model.RoleTester},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	project1 := model.Project{Name: "Alpha", OwnerID: users[0].ID, AgileMode: model.AgileModeKanban}
	project2 := model.Project{Name: "Beta", OwnerID: users[0].ID, AgileMode: model.AgileModeScrum}
	project3 := model.Project{Name: "Gamma", OwnerID: users[3].ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&[]model.Project{project1, project2, project3}).Error; err != nil {
		t.Fatalf("create projects: %v", err)
	}
	var projects []model.Project
	if err := db.Order("id ASC").Find(&projects).Error; err != nil {
		t.Fatalf("reload projects: %v", err)
	}
	project1, project2, project3 = projects[0], projects[1], projects[2]

	members := []model.ProjectMember{
		{ProjectID: project1.ID, UserID: users[0].ID, RoleInProject: model.RoleProduct},
		{ProjectID: project1.ID, UserID: users[2].ID, RoleInProject: model.RoleDeveloper},
		{ProjectID: project2.ID, UserID: users[3].ID, RoleInProject: model.RoleDeveloper},
	}
	if err := db.Create(&members).Error; err != nil {
		t.Fatalf("create members: %v", err)
	}
	if err := db.Create(&[]model.ProjectTechLead{
		{ProjectID: project1.ID, UserID: users[3].ID},
		{ProjectID: project3.ID, UserID: users[3].ID},
	}).Error; err != nil {
		t.Fatalf("create tech leads: %v", err)
	}

	story := model.UserStory{
		ProjectID:          project1.ID,
		Title:              "登录故事",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusPending,
		ReviewStatus:       model.ReviewStatusPending,
		CreatedBy:          users[0].ID,
		AcceptanceCriteria: datatypes.JSON([]byte("[]")),
		Tags:               datatypes.JSON([]byte("[]")),
		CodeReferences:     datatypes.JSON([]byte("[]")),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	return accessFixture{
		db:       db,
		owner:    users[0],
		admin:    users[1],
		member:   users[2],
		techLead: users[3],
		outsider: users[4],
		project1: project1,
		project2: project2,
		project3: project3,
		story:    story,
	}
}

func TestEnsureProjectAccess(t *testing.T) {
	fixture := newAccessFixture(t)

	tests := []struct {
		name      string
		userID    uint
		wantOwner bool
		wantErr   error
	}{
		{name: "owner", userID: fixture.owner.ID, wantOwner: true},
		{name: "admin", userID: fixture.admin.ID, wantOwner: false},
		{name: "member", userID: fixture.member.ID, wantOwner: false},
		{name: "tech lead", userID: fixture.techLead.ID, wantOwner: false},
		{name: "outsider", userID: fixture.outsider.ID, wantErr: ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, isOwner, err := EnsureProjectAccess(fixture.db, fixture.project1.ID, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if project.ID != fixture.project1.ID {
				t.Fatalf("expected project %d, got %d", fixture.project1.ID, project.ID)
			}
			if isOwner != tt.wantOwner {
				t.Fatalf("expected isOwner=%v, got %v", tt.wantOwner, isOwner)
			}
		})
	}
}

func TestEnsureStoryAccessAndReviewPermissions(t *testing.T) {
	fixture := newAccessFixture(t)

	story, project, isOwner, err := EnsureStoryAccess(fixture.db, fixture.story.ID, fixture.owner.ID)
	if err != nil {
		t.Fatalf("owner ensure story access: %v", err)
	}
	if story.ID != fixture.story.ID || project.ID != fixture.project1.ID || !isOwner {
		t.Fatalf("unexpected owner access result")
	}

	canReview, err := CanReviewStory(fixture.db, &fixture.story, fixture.admin.ID, fixture.admin.Role)
	if err != nil || !canReview {
		t.Fatalf("admin should review story, canReview=%v err=%v", canReview, err)
	}

	canReview, err = CanReviewStory(fixture.db, &fixture.story, fixture.techLead.ID, fixture.techLead.Role)
	if err != nil || !canReview {
		t.Fatalf("assigned tech lead should review story, canReview=%v err=%v", canReview, err)
	}

	canReview, err = CanReviewStory(fixture.db, &fixture.story, fixture.member.ID, fixture.member.Role)
	if err != nil {
		t.Fatalf("developer review err: %v", err)
	}
	if canReview {
		t.Fatalf("developer should not review story")
	}
}

func TestAccessibleProjectIDs(t *testing.T) {
	fixture := newAccessFixture(t)

	adminIDs, err := AccessibleProjectIDs(fixture.db, fixture.admin.ID, fixture.admin.Role)
	if err != nil {
		t.Fatalf("admin accessible ids: %v", err)
	}
	if len(adminIDs) != 3 || adminIDs[0] != fixture.project1.ID || adminIDs[2] != fixture.project3.ID {
		t.Fatalf("unexpected admin project ids: %v", adminIDs)
	}

	techLeadIDs, err := AccessibleProjectIDs(fixture.db, fixture.techLead.ID, fixture.techLead.Role)
	if err != nil {
		t.Fatalf("tech lead accessible ids: %v", err)
	}
	if len(techLeadIDs) != 3 {
		t.Fatalf("expected 3 project ids, got %v", techLeadIDs)
	}
	if techLeadIDs[0] != fixture.project1.ID || techLeadIDs[1] != fixture.project2.ID || techLeadIDs[2] != fixture.project3.ID {
		t.Fatalf("unexpected tech lead ids: %v", techLeadIDs)
	}

	memberIDs, err := AccessibleProjectIDs(fixture.db, fixture.member.ID, fixture.member.Role)
	if err != nil {
		t.Fatalf("member accessible ids: %v", err)
	}
	if len(memberIDs) != 1 || memberIDs[0] != fixture.project1.ID {
		t.Fatalf("unexpected member ids: %v", memberIDs)
	}
}
