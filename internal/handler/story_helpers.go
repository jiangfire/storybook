package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *StoryHandler) ensureProjectMember(projectID, userID uint) error {
	err := h.storySvc.EnsureProjectMember(projectID, userID)
	if errors.Is(err, service.ErrForbidden) {
		return errForbidden
	}
	return err
}

func normalizeAC(items []createStoryACItem) []model.AcceptanceCriterion {
	if len(items) == 0 {
		return []model.AcceptanceCriterion{}
	}

	normalized := make([]model.AcceptanceCriterion, 0, len(items))
	for i, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fmt.Sprintf("ac-%d", i+1)
		}

		order := item.Order
		if order <= 0 {
			order = i + 1
		}

		normalized = append(normalized, model.AcceptanceCriterion{
			ID:          id,
			Ref:         strings.TrimSpace(item.Ref),
			Description: strings.TrimSpace(item.Description),
			Status:      model.ACStatusPending,
			Order:       order,
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].Order < normalized[j].Order
	})

	return normalized
}

func summarizeAC(items []model.AcceptanceCriterion) (total, passed, pending, failed int) {
	total = len(items)
	for _, ac := range items {
		switch ac.Status {
		case model.ACStatusPassed:
			passed++
		case model.ACStatusFailed:
			failed++
		default:
			pending++
		}
	}
	return
}

func actorFromContext(c *gin.Context, userID uint) gin.H {
	return gin.H{
		"id":    userID,
		"email": c.GetString(middleware.CtxEmailKey),
	}
}

func denyTechLeadStoryMutation(c *gin.Context, role string) bool {
	if role == model.RoleTechLead {
		api.Forbidden(c, "技术负责人仅可审批用户故事")
		return true
	}
	return false
}

func parseStringArrayJSON(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	return out
}

func defaultColumnName(position int) string {
	switch position {
	case 0:
		return "待审批"
	case 1:
		return "待办"
	case 2:
		return "就绪"
	case 3:
		return "开发中"
	case 4:
		return "测试中"
	case 5:
		return "已完成"
	default:
		return "未命名列"
	}
}
