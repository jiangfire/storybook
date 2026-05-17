package e2e_test

import (
	"net/http"
	"strconv"
	"testing"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestTechLeadScopedAccessAssignAndReviewE2E(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(&config.Config{DBDriver: "sqlite", DBDSN: "file:e2e_techlead_global_read?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	tm := auth.NewTokenManager("e2e-secret-techlead-read", 24, 24*7)
	r := newTestEngine(t, db, tm)

	pm1ID := seedUserOnly(t, db, "pm1-techlead-e2e@example.com", model.RoleProduct)
	pm2ID := seedUserOnly(t, db, "pm2-techlead-e2e@example.com", model.RoleProduct)
	techLeadID := seedUserOnly(t, db, "techlead-global-e2e@example.com", model.RoleTechLead)
	devID := seedUserOnly(t, db, "dev-techlead-e2e@example.com", model.RoleDeveloper)

	pm1Token, _, _ := tm.GenerateAccessToken(pm1ID, "pm1-techlead-e2e@example.com", model.RoleProduct)
	pm2Token, _, _ := tm.GenerateAccessToken(pm2ID, "pm2-techlead-e2e@example.com", model.RoleProduct)
	techLeadToken, _, _ := tm.GenerateAccessToken(techLeadID, "techlead-global-e2e@example.com", model.RoleTechLead)

	project1Resp := doJSON(t, r, http.MethodPost, "/api/projects", pm1Token, map[string]any{
		"name":       "TechLead-Project-1",
		"agile_mode": "kanban",
	})
	if project1Resp.Code != http.StatusOK {
		t.Fatalf("create project1 failed: %d %s", project1Resp.Code, project1Resp.Body)
	}
	project1ID := uint(nestedFloat(t, project1Resp.JSON, "data", "id"))

	project2Resp := doJSON(t, r, http.MethodPost, "/api/projects", pm2Token, map[string]any{
		"name":       "TechLead-Project-2",
		"agile_mode": "kanban",
	})
	if project2Resp.Code != http.StatusOK {
		t.Fatalf("create project2 failed: %d %s", project2Resp.Code, project2Resp.Body)
	}
	project2ID := uint(nestedFloat(t, project2Resp.JSON, "data", "id"))

	if err := db.Create(&model.ProjectMember{
		ProjectID:     project2ID,
		UserID:        devID,
		RoleInProject: model.RoleDeveloper,
	}).Error; err != nil {
		t.Fatalf("add developer to project2 failed: %v", err)
	}

	story2Resp := doJSON(t, r, http.MethodPost, "/api/projects/"+strconv.Itoa(int(project2ID))+"/stories", pm2Token, map[string]any{
		"title":      "Story-Need-Review",
		"story_type": "feature",
		"priority":   2,
	})
	if story2Resp.Code != http.StatusOK {
		t.Fatalf("create story2 failed: %d %s", story2Resp.Code, story2Resp.Body)
	}
	story2ID := uint(nestedFloat(t, story2Resp.JSON, "data", "id"))

	if err := db.Create(&model.ProjectTechLead{
		ProjectID:  project2ID,
		UserID:     techLeadID,
		AssignedAt: db.NowFunc(),
	}).Error; err != nil {
		t.Fatalf("assign tech lead to project2 failed: %v", err)
	}

	listResp := doJSON(t, r, http.MethodGet, "/api/projects", techLeadToken, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("tech_lead list projects failed: %d %s", listResp.Code, listResp.Body)
	}
	if projectListContainsID(t, listResp.JSON, project1ID) || !projectListContainsID(t, listResp.JSON, project2ID) {
		t.Fatalf("tech_lead should only see assigned/participated projects, got: %s", listResp.Body)
	}

	getProjectResp := doJSON(t, r, http.MethodGet, "/api/projects/"+strconv.Itoa(int(project1ID)), techLeadToken, nil)
	if getProjectResp.Code != http.StatusForbidden {
		t.Fatalf("tech_lead get unassigned project should be forbidden, got: %d %s", getProjectResp.Code, getProjectResp.Body)
	}

	getStoryResp := doJSON(t, r, http.MethodGet, "/api/stories/"+strconv.Itoa(int(story2ID)), techLeadToken, nil)
	if getStoryResp.Code != http.StatusOK {
		t.Fatalf("tech_lead get story2 failed: %d %s", getStoryResp.Code, getStoryResp.Body)
	}

	userAdminResp := doJSON(t, r, http.MethodGet, "/api/admin/users", techLeadToken, nil)
	if userAdminResp.Code != http.StatusForbidden {
		t.Fatalf("tech_lead should not access admin user management, got: %d %s", userAdminResp.Code, userAdminResp.Body)
	}

	updateStatusResp := doJSON(t, r, http.MethodPatch, "/api/stories/"+strconv.Itoa(int(story2ID))+"/status", techLeadToken, map[string]any{
		"status": "backlog",
	})
	if updateStatusResp.Code != http.StatusForbidden {
		t.Fatalf("tech_lead update story status should be forbidden, got: %d %s", updateStatusResp.Code, updateStatusResp.Body)
	}

	assignResp := doJSON(t, r, http.MethodPatch, "/api/stories/"+strconv.Itoa(int(story2ID))+"/assignee", techLeadToken, map[string]any{
		"assigned_to": devID,
	})
	if assignResp.Code != http.StatusOK {
		t.Fatalf("tech_lead assign story failed: %d %s", assignResp.Code, assignResp.Body)
	}
	if uint(nestedFloat(t, assignResp.JSON, "data", "assigned_to", "id")) != devID {
		t.Fatalf("tech_lead assign story should set assignee to developer, got: %s", assignResp.Body)
	}

	updateStoryResp := doJSON(t, r, http.MethodPut, "/api/stories/"+strconv.Itoa(int(story2ID)), techLeadToken, map[string]any{
		"title": "edited-by-techlead",
	})
	if updateStoryResp.Code != http.StatusForbidden {
		t.Fatalf("tech_lead update story should be forbidden, got: %d %s", updateStoryResp.Code, updateStoryResp.Body)
	}

	reviewResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(story2ID))+"/review", techLeadToken, map[string]any{
		"approved": true,
		"comment":  "通过",
	})
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("tech_lead review story failed: %d %s", reviewResp.Code, reviewResp.Body)
	}
	if nestedString(t, reviewResp.JSON, "data", "status") != model.StoryStatusBacklog {
		t.Fatalf("approve should move story to backlog, got: %s", reviewResp.Body)
	}
}

func TestTechLeadRejectReviewRulesE2E(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(&config.Config{DBDriver: "sqlite", DBDSN: "file:e2e_techlead_review_rules?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	tm := auth.NewTokenManager("e2e-secret-techlead-review", 24, 24*7)
	r := newTestEngine(t, db, tm)

	pmID := seedUserOnly(t, db, "pm-review-e2e@example.com", model.RoleProduct)
	techLeadID := seedUserOnly(t, db, "techlead-review-e2e@example.com", model.RoleTechLead)

	pmToken, _, _ := tm.GenerateAccessToken(pmID, "pm-review-e2e@example.com", model.RoleProduct)
	techLeadToken, _, _ := tm.GenerateAccessToken(techLeadID, "techlead-review-e2e@example.com", model.RoleTechLead)

	projectResp := doJSON(t, r, http.MethodPost, "/api/projects", pmToken, map[string]any{
		"name":       "TechLead-Review-Project",
		"agile_mode": "kanban",
	})
	if projectResp.Code != http.StatusOK {
		t.Fatalf("create project failed: %d %s", projectResp.Code, projectResp.Body)
	}
	projectID := uint(nestedFloat(t, projectResp.JSON, "data", "id"))

	if err := db.Create(&model.ProjectTechLead{
		ProjectID:  projectID,
		UserID:     techLeadID,
		AssignedAt: db.NowFunc(),
	}).Error; err != nil {
		t.Fatalf("assign tech lead failed: %v", err)
	}

	storyResp := doJSON(t, r, http.MethodPost, "/api/projects/"+strconv.Itoa(int(projectID))+"/stories", pmToken, map[string]any{
		"title":      "Story-Reject-Required-Reason",
		"story_type": "feature",
		"priority":   1,
	})
	if storyResp.Code != http.StatusOK {
		t.Fatalf("create story failed: %d %s", storyResp.Code, storyResp.Body)
	}
	storyID := uint(nestedFloat(t, storyResp.JSON, "data", "id"))

	missingApprovedResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(storyID))+"/review", techLeadToken, map[string]any{
		"comment": "没有approved字段",
	})
	if missingApprovedResp.Code != http.StatusBadRequest {
		t.Fatalf("missing approved should be bad request, got: %d %s", missingApprovedResp.Code, missingApprovedResp.Body)
	}

	rejectNoReasonResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(storyID))+"/review", techLeadToken, map[string]any{
		"approved": false,
		"comment":  "   ",
	})
	if rejectNoReasonResp.Code != http.StatusBadRequest {
		t.Fatalf("reject without reason should be bad request, got: %d %s", rejectNoReasonResp.Code, rejectNoReasonResp.Body)
	}

	rejectReason := "验收标准不完整，需要补充边界场景"
	rejectResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(storyID))+"/review", techLeadToken, map[string]any{
		"approved": false,
		"comment":  rejectReason,
	})
	if rejectResp.Code != http.StatusOK {
		t.Fatalf("reject review failed: %d %s", rejectResp.Code, rejectResp.Body)
	}
	if nestedString(t, rejectResp.JSON, "data", "status") != model.StoryStatusPending {
		t.Fatalf("reject should keep story in pending, got: %s", rejectResp.Body)
	}
	if nestedString(t, rejectResp.JSON, "data", "review_status") != model.ReviewStatusRejected {
		t.Fatalf("reject should set review_status=rejected, got: %s", rejectResp.Body)
	}
	if nestedString(t, rejectResp.JSON, "data", "review_comment") != rejectReason {
		t.Fatalf("reject should persist reason, got: %s", rejectResp.Body)
	}

	getAfterRejectResp := doJSON(t, r, http.MethodGet, "/api/stories/"+strconv.Itoa(int(storyID)), techLeadToken, nil)
	if getAfterRejectResp.Code != http.StatusOK {
		t.Fatalf("get story after reject failed: %d %s", getAfterRejectResp.Code, getAfterRejectResp.Body)
	}
	if nestedString(t, getAfterRejectResp.JSON, "data", "status") != model.StoryStatusPending ||
		nestedString(t, getAfterRejectResp.JSON, "data", "review_status") != model.ReviewStatusRejected {
		t.Fatalf("story state after reject mismatch: %s", getAfterRejectResp.Body)
	}

	approveResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(storyID))+"/review", techLeadToken, map[string]any{
		"approved": true,
		"comment":  "可进入待办",
	})
	if approveResp.Code != http.StatusOK {
		t.Fatalf("approve after reject failed: %d %s", approveResp.Code, approveResp.Body)
	}
	if nestedString(t, approveResp.JSON, "data", "status") != model.StoryStatusBacklog {
		t.Fatalf("approve should move story to backlog, got: %s", approveResp.Body)
	}
	if nestedString(t, approveResp.JSON, "data", "review_status") != model.ReviewStatusApproved {
		t.Fatalf("approve should set review_status=approved, got: %s", approveResp.Body)
	}
	if nestedString(t, approveResp.JSON, "data", "review_comment") != "" {
		t.Fatalf("approve should clear review_comment, got: %s", approveResp.Body)
	}
}

func seedUserOnly(t *testing.T, db *gorm.DB, email, role string) uint {
	t.Helper()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), 10)
	user := model.User{
		Username:       email,
		Email:          email,
		HashedPassword: string(hashed),
		Role:           role,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	return user.ID
}

func projectListContainsID(t *testing.T, payload map[string]any, projectID uint) bool {
	t.Helper()
	items, ok := nestedValue(t, payload, "data", "projects").([]any)
	if !ok {
		t.Fatalf("projects field is not array: %#v", nestedValue(t, payload, "data", "projects"))
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		idRaw, ok := row["id"].(float64)
		if ok && uint(idRaw) == projectID {
			return true
		}
	}
	return false
}
