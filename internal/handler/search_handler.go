package handler

import (
	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SearchHandler struct {
	db          *gorm.DB
	vectorSvc   service.VectorService
	storyRepo   repository.StoryRepo
	bugRepo     repository.BugRepo
	projectRepo repository.ProjectRepo
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return NewSearchHandlerWithVector(db, nil)
}

// NewSearchHandlerWithVector 创建带向量搜索的 SearchHandler
func NewSearchHandlerWithVector(db *gorm.DB, vectorSvc service.VectorService) *SearchHandler {
	return &SearchHandler{
		db:          db,
		vectorSvc:   vectorSvc,
		storyRepo:   repository.NewStoryRepository(db),
		bugRepo:     repository.NewBugRepository(db),
		projectRepo: repository.NewProjectRepository(db),
	}
}

func (h *SearchHandler) Capabilities(c *gin.Context) {
	api.Success(c, "success", gin.H{
		"semantic_enabled": h.vectorSvc != nil,
		"filters": gin.H{
			"date_range":   []string{"created_from", "created_to"},
			"status_array": true,
			"assignee":     true,
		},
	})
}
