package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/config"
	"git.neolidy.top/neo/storybook/internal/database"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/router"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestAPIMainFlowE2E(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := "file:e2e_api_main_flow?mode=memory&cache=shared"
	db, err := database.Connect(&config.Config{DBDriver: "sqlite", DBDSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}

	tm := auth.NewTokenManager("e2e-secret", 24, 24*7)
	r := router.New(db, tm)

	// 1) 注册产品经理并拿token
	regResp := doJSON(t, r, http.MethodPost, "/api/auth/register", "", map[string]any{
		"email":    "pm-e2e@example.com",
		"password": "Pass1234",
		"role":     "product",
	})
	if regResp.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", regResp.Code, regResp.Body)
	}
	pmToken := nestedString(t, regResp.JSON, "data", "token")
	pmUserID := uint(nestedFloat(t, regResp.JSON, "data", "user", "id"))

	// 2) 创建项目
	projectResp := doJSON(t, r, http.MethodPost, "/api/projects", pmToken, map[string]any{
		"name":        "E2E Project",
		"description": "e2e flow",
		"agile_mode":  "kanban",
	})
	if projectResp.Code != http.StatusOK {
		t.Fatalf("create project failed: %d %s", projectResp.Code, projectResp.Body)
	}
	projectID := uint(nestedFloat(t, projectResp.JSON, "data", "id"))

	// 3) 创建故事
	storyResp := doJSON(t, r, http.MethodPost, "/api/projects/"+strconv.Itoa(int(projectID))+"/stories", pmToken, map[string]any{
		"title":        "E2E Story",
		"description":  "story desc",
		"story_type":   "feature",
		"priority":     2,
		"story_points": 3,
		"acceptance_criteria": []map[string]any{
			{"id": "ac-1", "description": "first ac", "order": 1},
			{"id": "ac-2", "description": "second ac", "order": 2},
		},
	})
	if storyResp.Code != http.StatusOK {
		t.Fatalf("create story failed: %d %s", storyResp.Code, storyResp.Body)
	}
	storyID := uint(nestedFloat(t, storyResp.JSON, "data", "id"))

	// 4) AC拆分子任务
	splitResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(storyID))+"/tasks/split-from-ac", pmToken, map[string]any{})
	if splitResp.Code != http.StatusOK {
		t.Fatalf("split tasks failed: %d %s", splitResp.Code, splitResp.Body)
	}

	// 5) 手工创建一个任务
	taskResp := doJSON(t, r, http.MethodPost, "/api/stories/"+strconv.Itoa(int(storyID))+"/tasks", pmToken, map[string]any{
		"title":           "Implement API",
		"description":     "build endpoint",
		"priority":        3,
		"estimated_hours": 6,
	})
	if taskResp.Code != http.StatusOK {
		t.Fatalf("create task failed: %d %s", taskResp.Code, taskResp.Body)
	}
	taskID := uint(nestedFloat(t, taskResp.JSON, "data", "id"))

	// 6) 构造开发、测试用户并加入项目
	devID := seedUserAndMember(t, db, projectID, "dev-e2e@example.com", model.RoleDeveloper)
	testerID := seedUserAndMember(t, db, projectID, "tester-e2e@example.com", model.RoleTester)

	devToken, _, _ := tm.GenerateAccessToken(devID, "dev-e2e@example.com", model.RoleDeveloper)
	testerToken, _, _ := tm.GenerateAccessToken(testerID, "tester-e2e@example.com", model.RoleTester)

	// 7) 开发领取任务并推进进度
	claimResp := doJSON(t, r, http.MethodPost, "/api/tasks/"+strconv.Itoa(int(taskID))+"/claim", devToken, nil)
	if claimResp.Code != http.StatusOK {
		t.Fatalf("claim task failed: %d %s", claimResp.Code, claimResp.Body)
	}

	progressResp := doJSON(t, r, http.MethodPatch, "/api/tasks/"+strconv.Itoa(int(taskID))+"/progress", devToken, map[string]any{"progress": 100})
	if progressResp.Code != http.StatusOK {
		t.Fatalf("update progress failed: %d %s", progressResp.Code, progressResp.Body)
	}
	if nestedString(t, progressResp.JSON, "data", "status") != "done" {
		t.Fatalf("expected task status done, got %s", nestedString(t, progressResp.JSON, "data", "status"))
	}

	// 8) 测试创建缺陷
	bugResp := doJSON(t, r, http.MethodPost, "/api/projects/"+strconv.Itoa(int(projectID))+"/bugs", testerToken, map[string]any{
		"story_id":    storyID,
		"title":       "E2E Bug",
		"description": "bug desc",
		"severity":    "high",
		"assigned_to": devID,
	})
	if bugResp.Code != http.StatusOK {
		t.Fatalf("create bug failed: %d %s", bugResp.Code, bugResp.Body)
	}
	bugID := uint(nestedFloat(t, bugResp.JSON, "data", "id"))

	bugStatusResp := doJSON(t, r, http.MethodPatch, "/api/bugs/"+strconv.Itoa(int(bugID))+"/status", devToken, map[string]any{"status": "resolved"})
	if bugStatusResp.Code != http.StatusOK {
		t.Fatalf("resolve bug failed: %d %s", bugStatusResp.Code, bugStatusResp.Body)
	}

	// 9) 项目报表
	qualityResp := doJSON(t, r, http.MethodGet, "/api/projects/"+strconv.Itoa(int(projectID))+"/reports/quality", pmToken, nil)
	if qualityResp.Code != http.StatusOK {
		t.Fatalf("quality report failed: %d %s", qualityResp.Code, qualityResp.Body)
	}

	// 10) 创建冲刺并规划故事，检查燃尽图接口
	start := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	end := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	sprintResp := doJSON(t, r, http.MethodPost, "/api/projects/"+strconv.Itoa(int(projectID))+"/sprints", pmToken, map[string]any{
		"name":       "Sprint-E2E",
		"goal":       "goal",
		"start_date": start,
		"end_date":   end,
	})
	if sprintResp.Code != http.StatusOK {
		t.Fatalf("create sprint failed: %d %s", sprintResp.Code, sprintResp.Body)
	}
	sprintID := uint(nestedFloat(t, sprintResp.JSON, "data", "id"))

	assignStoryResp := doJSON(t, r, http.MethodPatch, "/api/stories/"+strconv.Itoa(int(storyID))+"/sprint", pmToken, map[string]any{"sprint_id": sprintID})
	if assignStoryResp.Code != http.StatusOK {
		t.Fatalf("assign story sprint failed: %d %s", assignStoryResp.Code, assignStoryResp.Body)
	}

	burndownResp := doJSON(t, r, http.MethodGet, "/api/projects/"+strconv.Itoa(int(projectID))+"/reports/burndown?sprint_id="+strconv.Itoa(int(sprintID)), pmToken, nil)
	if burndownResp.Code != http.StatusOK {
		t.Fatalf("burndown report failed: %d %s", burndownResp.Code, burndownResp.Body)
	}

	_ = pmUserID
}

type httpResult struct {
	Code int
	Body string
	JSON map[string]any
}

func doJSON(t *testing.T, r http.Handler, method, path, token string, body any) httpResult {
	t.Helper()
	var reqBody []byte
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal req body failed: %v", err)
		}
		reqBody = buf
	} else {
		reqBody = []byte("{}")
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := httpResult{Code: w.Code, Body: w.Body.String(), JSON: map[string]any{}}
	_ = json.Unmarshal(w.Body.Bytes(), &res.JSON)
	return res
}

func nestedFloat(t *testing.T, payload map[string]any, keys ...string) float64 {
	t.Helper()
	value := nestedValue(t, payload, keys...)
	f, ok := value.(float64)
	if !ok {
		t.Fatalf("value at %v is not float64: %#v", keys, value)
	}
	return f
}

func nestedString(t *testing.T, payload map[string]any, keys ...string) string {
	t.Helper()
	value := nestedValue(t, payload, keys...)
	s, ok := value.(string)
	if !ok {
		t.Fatalf("value at %v is not string: %#v", keys, value)
	}
	return s
}

func nestedValue(t *testing.T, payload map[string]any, keys ...string) any {
	t.Helper()
	cur := any(payload)
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %v invalid, %T is not map", keys, cur)
		}
		cur, ok = m[k]
		if !ok {
			t.Fatalf("key %s not found in path %v", k, keys)
		}
	}
	return cur
}

func seedUserAndMember(t *testing.T, db *gorm.DB, projectID uint, email, role string) uint {
	t.Helper()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), 10)
	user := model.User{
		Username:       strings.Split(email, "@")[0],
		Email:          email,
		HashedPassword: string(hashed),
		Role:           role,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	member := model.ProjectMember{ProjectID: projectID, UserID: user.ID, RoleInProject: role}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("seed member failed: %v", err)
	}
	return user.ID
}
