package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.neolidy.top/neo/storybook/internal/model"
)

// parseStoryFromContent attempts to extract a StoryResult from raw LLM output.
// Falls back to heuristic decomposition if JSON parsing fails.
func parseStoryFromContent(content, fallbackRequirement string) (*StoryResult, error) {
	// Try to find JSON in the response
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		// No JSON found; use heuristic fallback
		h := &heuristicAIService{}
		return h.GenerateStory(context.Background(), fallbackRequirement)
	}

	var result StoryResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		h := &heuristicAIService{}
		return h.GenerateStory(context.Background(), fallbackRequirement)
	}

	// Validate minimum fields
	if strings.TrimSpace(result.Actor) == "" || strings.TrimSpace(result.Action) == "" {
		h := &heuristicAIService{}
		return h.GenerateStory(context.Background(), fallbackRequirement)
	}

	result.Actor = strings.TrimSpace(result.Actor)
	result.Action = strings.TrimSpace(result.Action)
	result.Value = strings.TrimSpace(result.Value)
	result.Title = normalizeTitle(result.Title, fallbackRequirement, result.Action)
	result.StoryType = normalizeStoryType(result.StoryType, fallbackRequirement)
	result.Priority = normalizePriority(result.Priority, fallbackRequirement)
	result.SuggestedAC = normalizeStringList(result.SuggestedAC, 8)
	result.Tags = normalizeTags(result.Tags, fallbackRequirement)
	result.Warnings = normalizeStringList(result.Warnings, 3)

	if strings.TrimSpace(result.UserStory) == "" {
		result.UserStory = fmt.Sprintf("作为 %s，\n我想要 %s，\n以便 %s", result.Actor, result.Action, normalizeValue(result.Value))
	}
	if strings.TrimSpace(result.Value) == "" {
		result.Value = inferValue(fallbackRequirement)
	}
	if result.UserStory == "" {
		result.UserStory = fmt.Sprintf("作为 %s，\n我想要 %s，\n以便 %s", result.Actor, result.Action, result.Value)
	}
	if result.StoryPoints <= 0 {
		result.StoryPoints = EstimatePoints(result.Action, len(result.SuggestedAC))
	}
	result.StoryPoints = normalizeStoryPoints(result.StoryPoints)
	result.Source = AIResponseSourceOpenAI

	return &result, nil
}

// extractJSON finds the first {...} block in content.
func extractJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	if start < 0 {
		return ""
	}
	// Find matching closing brace
	depth := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}
	return ""
}

func normalizeTitle(title, requirement, action string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = inferTitle(requirement, action)
	}
	if len([]rune(title)) > 200 {
		title = string([]rune(title)[:200])
	}
	return title
}

func normalizeStoryType(storyType, requirement string) string {
	switch strings.TrimSpace(strings.ToLower(storyType)) {
	case model.StoryTypeFeature, model.StoryTypeBug, model.StoryTypeChore:
		return strings.TrimSpace(strings.ToLower(storyType))
	default:
		return inferStoryType(requirement)
	}
}

func normalizePriority(priority int, requirement string) int {
	if priority < 0 || priority > 4 {
		return inferPriority(requirement)
	}
	return priority
}

func normalizeStoryPoints(points int) int {
	allowed := []int{1, 2, 3, 5, 8, 13}
	for _, item := range allowed {
		if points == item {
			return points
		}
	}

	if points <= 1 {
		return 1
	}
	if points <= 2 {
		return 2
	}
	if points <= 3 {
		return 3
	}
	if points <= 5 {
		return 5
	}
	if points <= 8 {
		return 8
	}
	return 13
}

func normalizeStringList(items []string, maxCount int) []string {
	if len(items) == 0 {
		return nil
	}

	normalized := make([]string, 0, min(len(items), maxCount))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
		if len(normalized) == maxCount {
			break
		}
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeTags(tags []string, requirement string) []string {
	normalized := normalizeStringList(tags, 5)
	if len(normalized) > 0 {
		return normalized
	}
	return inferTags(requirement)
}

func normalizeValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "提升任务交付效率"
	}
	return value
}
