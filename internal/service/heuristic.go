package service

import (
	"strings"

	"github.com/jiangfire/storybook/internal/model"
)

// inferActor 从需求文本中推断用户角色。
func inferActor(text string) string {
	candidates := []string{"产品经理", "开发人员", "测试人员", "用户", "访客", "买家", "卖家", "管理员"}
	for _, c := range candidates {
		if strings.Contains(text, c) {
			return c
		}
	}
	return "用户"
}

// inferAction 提取或截断需求文本作为动作描述。
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

// inferValue 从需求文本中提取"以便"后的价值描述。
func inferValue(text string) string {
	if strings.Contains(text, "以便") {
		parts := strings.SplitN(text, "以便", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return "提升任务交付效率"
}

// inferTitle 从动作描述和需求文本推断故事标题。
func inferTitle(text, action string) string {
	title := strings.TrimSpace(action)
	if title == "" {
		title = strings.TrimSpace(text)
	}
	if title == "" {
		return "未命名故事"
	}
	title = strings.TrimPrefix(title, "作为")
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}
	return title
}

// inferStoryType 根据关键词推断故事类型。
func inferStoryType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "bug"), strings.Contains(lower, "缺陷"), strings.Contains(lower, "报错"), strings.Contains(lower, "修复"):
		return model.StoryTypeBug
	case strings.Contains(lower, "清理"), strings.Contains(lower, "重构"), strings.Contains(lower, "迁移"), strings.Contains(lower, "优化脚本"):
		return model.StoryTypeChore
	default:
		return model.StoryTypeFeature
	}
}

// inferPriority 根据关键词推断优先级(1-4)。
func inferPriority(text string) int {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "紧急"), strings.Contains(lower, "阻塞"), strings.Contains(lower, "critical"), strings.Contains(lower, "p0"):
		return 4
	case strings.Contains(lower, "高优"), strings.Contains(lower, "high"), strings.Contains(lower, "p1"):
		return 3
	case strings.Contains(lower, "低优"), strings.Contains(lower, "low"), strings.Contains(lower, "p3"):
		return 1
	default:
		return 2
	}
}

// inferTags 根据关键词匹配推断故事标签，最多返回5个。
func inferTags(text string) []string {
	candidates := []struct {
		keyword string
		tag     string
	}{
		{keyword: "登录", tag: "登录"},
		{keyword: "认证", tag: "认证"},
		{keyword: "支付", tag: "支付"},
		{keyword: "订单", tag: "订单"},
		{keyword: "搜索", tag: "搜索"},
		{keyword: "消息", tag: "消息"},
		{keyword: "权限", tag: "权限"},
		{keyword: "报表", tag: "报表"},
		{keyword: "导出", tag: "导出"},
		{keyword: "后台", tag: "后台"},
		{keyword: "移动端", tag: "移动端"},
	}

	tags := make([]string, 0, 5)
	for _, candidate := range candidates {
		if strings.Contains(text, candidate.keyword) {
			tags = append(tags, candidate.tag)
		}
		if len(tags) == 5 {
			break
		}
	}
	return tags
}

// inferWarnings 根据需求文本特征生成 AI 提示警告。
func inferWarnings(text string) []string {
	warnings := make([]string, 0, 2)
	if len([]rune(text)) > 300 {
		warnings = append(warnings, "需求描述较长，建议人工确认是否需要拆分为多个故事")
	}
	if !strings.Contains(text, "用户") && !strings.Contains(text, "管理员") && !strings.Contains(text, "角色") {
		warnings = append(warnings, "需求中未明确角色，AI 已按通用用户角色推断")
	}
	return warnings
}
