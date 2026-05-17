package handler

import (
	"strconv"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/repository"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	repo *repository.NotificationRepository
}

func NewNotificationHandler(repo *repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := middleware.MustUserID(c)

	unreadOnly := strings.EqualFold(strings.TrimSpace(c.DefaultQuery("unread_only", "false")), "true")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	items, total, err := h.repo.ListByUser(userID, unreadOnly, page, limit)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	payload := make([]gin.H, 0, len(items))
	for _, n := range items {
		row := gin.H{
			"id":          n.ID,
			"type":        n.Type,
			"entity_type": n.EntityType,
			"entity_id":   n.EntityID,
			"project_id":  n.ProjectID,
			"title":       n.Title,
			"body":        n.Body,
			"metadata":    n.Metadata,
			"read_at":     n.ReadAt,
			"created_at":  n.CreatedAt,
		}
		if n.Actor != nil {
			row["actor"] = gin.H{
				"id":    n.Actor.ID,
				"email": n.Actor.Email,
			}
		}
		payload = append(payload, row)
	}

	api.Success(c, "success", gin.H{
		"items":       payload,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"unread_only": unreadOnly,
	})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := middleware.MustUserID(c)

	count, err := h.repo.UnreadCount(userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	api.Success(c, "success", gin.H{"count": count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := middleware.MustUserID(c)

	notifID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "通知ID无效")
		return
	}

	changed, err := h.repo.MarkRead(notifID, userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if !changed {
		// Either the notification doesn't belong to the caller or it's already
		// read. Both are treated as a 404 to avoid leaking ID existence.
		api.NotFound(c, "通知不存在或已读")
		return
	}

	api.Success(c, "success", gin.H{"id": notifID})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := middleware.MustUserID(c)

	count, err := h.repo.MarkAllRead(userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	api.Success(c, "success", gin.H{"updated": count})
}
