package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/api"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/repository"
	"github.com/jiangfire/storybook/internal/service"
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

// AssigneeCandidates 列出调用者可访问项目的全部成员与 Owner（去重），
// 供搜索页负责人过滤下拉使用；无可见项目时返回空列表。
func (h *SearchHandler) AssigneeCandidates(c *gin.Context) {
	userID := middleware.MustUserID(c)
	role, _ := middleware.CurrentRole(c)

	users := make([]gin.H, 0)
	projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if len(projectIDs) == 0 {
		api.Success(c, "success", gin.H{"users": users})
		return
	}

	var rows []model.User
	if err := h.db.Model(&model.User{}).
		Where("id IN (SELECT user_id FROM project_members WHERE project_id IN ? AND deleted_at IS NULL)", projectIDs).
		Or("id IN (SELECT owner_id FROM projects WHERE id IN ?)", projectIDs).
		Order("email ASC").
		Find(&rows).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	for _, u := range rows {
		users = append(users, gin.H{"id": u.ID, "email": u.Email})
	}
	api.Success(c, "success", gin.H{"users": users})
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
