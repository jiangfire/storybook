package handler

import (
	"fmt"
	"math"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIHandler struct {
	db *gorm.DB
}

func NewAIHandler(db *gorm.DB) *AIHandler {
	return &AIHandler{db: db}
}

type generateStoryRequest struct {
	Requirement string `json:"requirement" binding:"required,min=5,max=2000"`
}

func (h *AIHandler) GenerateStory(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可使用AI拆解")
		return
	}

	var req generateStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	reqText := strings.TrimSpace(req.Requirement)
	actor := inferActor(reqText)
	action := inferAction(reqText)
	value := inferValue(reqText)

	story := fmt.Sprintf("作为 %s，\n我想要 %s，\n以便 %s", actor, action, value)
	ac := []string{
		"支持核心输入与校验流程",
		"操作成功后返回明确反馈",
		"异常场景有清晰错误提示",
		"关键操作记录活动日志",
	}

	estimatedPoints := estimatePoints(reqText, len(ac))

	api.Success(c, "success", gin.H{
		"user_story":   story,
		"actor":        actor,
		"action":       action,
		"value":        value,
		"suggested_ac": ac,
		"story_points": estimatedPoints,
	})
}

type splitStoryReq struct {
	TargetCount int `json:"target_count"`
}

func (h *AIHandler) SplitStory(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可使用AI拆分")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	var req splitStoryReq
	if c.Request.ContentLength > 0 {
		if !middleware.BindJSON(c, &req) {
			return
		}
	}
	if req.TargetCount <= 0 || req.TargetCount > 8 {
		req.TargetCount = 3
	}

	var story model.UserStory
	if err := h.db.First(&story, storyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		api.Internal(c, "AC数据损坏")
		return
	}

	if len(criteria) == 0 {
		criteria = []model.AcceptanceCriterion{{ID: "ac-1", Description: story.Title, Status: model.ACStatusPending, Order: 1}}
	}

	count := req.TargetCount
	if count > len(criteria) {
		count = len(criteria)
	}
	if count <= 0 {
		count = 1
	}

	chunks := chunkCriteria(criteria, count)
	subStories := make([]gin.H, 0, len(chunks))
	for i, chunk := range chunks {
		title := fmt.Sprintf("%s - 子故事%d", story.Title, i+1)
		subStories = append(subStories, gin.H{
			"title":               title,
			"description":         fmt.Sprintf("由原故事 #%d 拆分", story.ID),
			"acceptance_criteria": chunk,
			"story_points":        estimatePoints(title, len(chunk)),
		})
	}

	api.Success(c, "success", gin.H{
		"story_id":     story.ID,
		"origin_title": story.Title,
		"sub_stories":  subStories,
	})
}

func (h *AIHandler) INVESTCheck(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可使用INVEST检查")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	var story model.UserStory
	if err := h.db.First(&story, storyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	contentLen := len([]rune(strings.TrimSpace(story.Description + " " + story.Title)))
	acCount := len(criteria)

	independent := scoreIndependent(story.Title)
	negotiable := scoreNegotiable(story.Description)
	valuable := scoreValuable(story.Description)
	estimable := scoreEstimable(acCount, story.Points)
	small := scoreSmall(contentLen, acCount, story.Points)
	testable := scoreTestable(acCount)

	total := (independent + negotiable + valuable + estimable + small + testable) / 6
	total = math.Round(total*10) / 10

	suggestions := make([]string, 0)
	if small < 0.6 {
		suggestions = append(suggestions, "故事过大，建议拆分为2-4个子故事")
	}
	if testable < 0.7 {
		suggestions = append(suggestions, "建议补充可执行的验收标准（AC）")
	}
	if negotiable < 0.7 {
		suggestions = append(suggestions, "描述过于刚性，建议减少实现细节以保留协商空间")
	}

	api.Success(c, "success", gin.H{
		"story_id":     story.ID,
		"invest_score": total,
		"checks": gin.H{
			"independent": checkPayload(independent, "故事独立性", "评估故事是否依赖其他故事"),
			"negotiable":  checkPayload(negotiable, "可协商性", "评估描述是否过于限制实现方式"),
			"valuable":    checkPayload(valuable, "价值性", "评估是否体现用户价值"),
			"estimable":   checkPayload(estimable, "可估算性", "评估是否可进行工作量估算"),
			"small":       checkPayload(small, "小粒度", "评估是否需要拆分"),
			"testable":    checkPayload(testable, "可测试性", "评估验收标准是否充分"),
		},
		"suggestions": suggestions,
	})
}

func inferActor(text string) string {
	candidates := []string{"产品经理", "开发人员", "测试人员", "用户", "访客", "买家", "卖家", "管理员"}
	for _, c := range candidates {
		if strings.Contains(text, c) {
			return c
		}
	}
	return "用户"
}

func inferAction(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "用户") || strings.HasPrefix(text, "作为") {
		return text
	}
	if len([]rune(text)) > 40 {
		return string([]rune(text)[:40]) + "..."
	}
	return text
}

func inferValue(text string) string {
	if strings.Contains(text, "以便") {
		parts := strings.SplitN(text, "以便", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return "提升任务交付效率"
}

func estimatePoints(text string, acCount int) int {
	l := len([]rune(text))
	score := l/120 + acCount/2
	switch {
	case score <= 1:
		return 1
	case score <= 2:
		return 2
	case score <= 3:
		return 3
	case score <= 5:
		return 5
	case score <= 8:
		return 8
	default:
		return 13
	}
}

func chunkCriteria(criteria []model.AcceptanceCriterion, chunkCount int) [][]model.AcceptanceCriterion {
	if chunkCount <= 1 {
		return [][]model.AcceptanceCriterion{criteria}
	}
	chunks := make([][]model.AcceptanceCriterion, chunkCount)
	for i, ac := range criteria {
		idx := i % chunkCount
		chunks[idx] = append(chunks[idx], ac)
	}
	out := make([][]model.AcceptanceCriterion, 0, chunkCount)
	for _, ch := range chunks {
		if len(ch) > 0 {
			out = append(out, ch)
		}
	}
	return out
}

func scoreIndependent(title string) float64 {
	if strings.Contains(title, "和") || strings.Contains(strings.ToLower(title), "and") {
		return 0.6
	}
	return 0.8
}

func scoreNegotiable(desc string) float64 {
	if len([]rune(desc)) > 500 {
		return 0.6
	}
	return 0.8
}

func scoreValuable(desc string) float64 {
	keywords := []string{"价值", "效率", "转化", "体验", "质量", "收益"}
	for _, k := range keywords {
		if strings.Contains(desc, k) {
			return 0.9
		}
	}
	if strings.TrimSpace(desc) == "" {
		return 0.5
	}
	return 0.7
}

func scoreEstimable(acCount int, points *int) float64 {
	if points != nil {
		return 0.85
	}
	if acCount >= 2 {
		return 0.75
	}
	return 0.55
}

func scoreSmall(contentLen, acCount int, points *int) float64 {
	if points != nil && *points >= 8 {
		return 0.45
	}
	if contentLen > 500 || acCount > 7 {
		return 0.5
	}
	if contentLen > 300 || acCount > 5 {
		return 0.65
	}
	return 0.85
}

func scoreTestable(acCount int) float64 {
	if acCount >= 5 {
		return 0.9
	}
	if acCount >= 3 {
		return 0.75
	}
	if acCount >= 1 {
		return 0.6
	}
	return 0.3
}

func checkPayload(score float64, title, desc string) gin.H {
	status := "pass"
	if score < 0.6 {
		status = "fail"
	} else if score < 0.75 {
		status = "warning"
	}
	return gin.H{
		"score":       score,
		"status":      status,
		"title":       title,
		"description": desc,
	}
}
