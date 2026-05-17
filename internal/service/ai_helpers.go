package service

import (
	"regexp"
	"strings"
)

// 本文件集中 ai_service.go 末尾的纯函数工具,
// ai_service.go 因此可以专注于 AIService 接口实现 (openAIService /
// heuristicAIService) 与 NewAIService 工厂。

// EstimatePoints 给出一个粗略的故事点估算,文本长度 + 验收数量两个变量都喂进
// 同一个梯度。openai / heuristic 两个 impl 与 handler 的 SplitStory 都依赖它,
// 故必须 export。
func EstimatePoints(text string, acCount int) int {
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

// SanitizeRequirement 把用户提交的需求文本压成单空格分隔的单行字符串,
// 防止换行/全角空格污染 prompt。
func SanitizeRequirement(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	spaceRegexp := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(spaceRegexp.ReplaceAllString(trimmed, " "))
}

// ResolveAIResponseSource 在 handler 没有更具体的来源信息时,从当前 AIService
// 实现类型推断 source。
func ResolveAIResponseSource(svc StoryGenerator) string {
	if svc != nil && svc.IsConfigured() {
		return AIResponseSourceOpenAI
	}
	return AIResponseSourceHeuristic
}

// ResolveStoryResultSource 优先采用 StoryResult.Source 显式标记,否则回退到
// 按 AIService 实现类型推断。
func ResolveStoryResultSource(result *StoryResult, fallbackSvc StoryGenerator) string {
	if result != nil && strings.TrimSpace(result.Source) != "" {
		return result.Source
	}
	return ResolveAIResponseSource(fallbackSvc)
}

// withOpenAIFallbackWarning 给非 openai 来源的结果追加一条统一文案,告知前端
// 这是规则草稿,需要人工再确认。
func withOpenAIFallbackWarning(result *StoryResult) *StoryResult {
	if result == nil || result.Source == AIResponseSourceOpenAI {
		return result
	}
	result.Warnings = PrependAIWarning(
		result.Warnings,
		"OpenAI 返回结果不可解析，已自动回退到规则草稿，请人工确认后再保存",
	)
	return result
}

// PrependAIWarning 把一条警告插到 warnings 列表的最前面,自动去重。
func PrependAIWarning(warnings []string, warning string) []string {
	warning = strings.TrimSpace(warning)
	if warning == "" {
		return warnings
	}
	for _, item := range warnings {
		if item == warning {
			return warnings
		}
	}
	return append([]string{warning}, warnings...)
}
