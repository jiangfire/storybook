package service

import (
	"errors"
)

// 向量服务相关错误
var (
	ErrEmptyText     = errors.New("text cannot be empty")
	ErrInvalidVector = errors.New("invalid vector dimension")
	ErrIndexFailed   = errors.New("indexing failed")
	ErrSearchFailed  = errors.New("search failed")
	ErrNoResults     = errors.New("no results found")
)

// SimilarStory 表示相似故事及其相似度分数
type SimilarStory struct {
	Story      any
	Similarity float64
}

// ProjectMatch 表示匹配的项目及其相关故事
type ProjectMatch struct {
	Project    any
	Similarity float64
	TopStories []SimilarStory
}
