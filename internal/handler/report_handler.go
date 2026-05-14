package handler

import (
	"errors"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReportHandler struct {
	db        *gorm.DB
	storyRepo *repository.StoryRepository
	sprintRepo *repository.SprintRepository
	taskRepo   *repository.TaskRepository
}

type burndownDoneRange struct {
	start time.Time
	end   time.Time
}

func NewReportHandler(db *gorm.DB) *ReportHandler {
	return &ReportHandler{
		db:         db,
		storyRepo:  repository.NewStoryRepository(db),
		sprintRepo: repository.NewSprintRepository(db),
		taskRepo:   repository.NewTaskRepository(db),
	}
}

func (h *ReportHandler) Velocity(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "项目不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	sprints, err := h.sprintRepo.ListByProject(projectID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(sprints))
	for _, s := range sprints {
		stories, err := h.storyRepo.ListBySprint(s.ID)
		if err != nil {
			continue
		}

		plannedPoints := 0
		donePoints := 0
		for _, st := range stories {
			p := 1
			if st.Points != nil {
				p = *st.Points
			}
			plannedPoints += p
			if st.Status == model.StoryStatusDone {
				donePoints += p
			}
		}

		items = append(items, gin.H{
			"sprint_id":        s.ID,
			"name":             s.Name,
			"status":           s.Status,
			"start_date":       s.StartDate,
			"end_date":         s.EndDate,
			"story_count":      len(stories),
			"planned_points":   plannedPoints,
			"completed_points": donePoints,
			"velocity": func() float64 {
				if plannedPoints == 0 {
					return 0
				}
				return float64(donePoints) / float64(plannedPoints) * 100
			}(),
		})
	}

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"velocity":   items,
	})
}

func (h *ReportHandler) Quality(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "项目不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	statusBreakdown := map[string]int64{}
	for _, st := range []string{model.BugStatusOpen, model.BugStatusInProgress, model.BugStatusResolved, model.BugStatusClosed} {
		var n int64
		logging.LogIfErr(h.db.Model(&model.BugReport{}).Where("project_id = ? AND status = ?", projectID, st).Count(&n).Error, "count bugs by status failed", "project_id", projectID, "status", st)
		statusBreakdown[st] = n
	}

	severityBreakdown := map[string]int64{}
	for _, sv := range []string{model.BugSeverityLow, model.BugSeverityMedium, model.BugSeverityHigh, model.BugSeverityCritical} {
		var n int64
		logging.LogIfErr(h.db.Model(&model.BugReport{}).Where("project_id = ? AND severity = ?", projectID, sv).Count(&n).Error, "count bugs by severity failed", "project_id", projectID, "severity", sv)
		severityBreakdown[sv] = n
	}

	var stories []model.UserStory
	logging.LogIfErr(h.db.Where("project_id = ?", projectID).Find(&stories).Error, "load project stories failed", "project_id", projectID)
	acTotal := 0
	acPassed := 0
	acFailed := 0
	for _, s := range stories {
		criteria, err := model.ParseAcceptanceCriteria(s.AcceptanceCriteria)
		if err != nil {
			continue
		}
		for _, ac := range criteria {
			acTotal++
			switch ac.Status {
			case model.ACStatusPassed:
				acPassed++
			case model.ACStatusFailed:
				acFailed++
			}
		}
	}

	acCompletion := 0.0
	if acTotal > 0 {
		acCompletion = float64(acPassed) / float64(acTotal) * 100
	}

	var totalBugs int64
	logging.LogIfErr(h.db.Model(&model.BugReport{}).Where("project_id = ?", projectID).Count(&totalBugs).Error, "count project bugs failed", "project_id", projectID)

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"bugs": gin.H{
			"total":              totalBugs,
			"status_breakdown":   statusBreakdown,
			"severity_breakdown": severityBreakdown,
		},
		"acceptance_criteria": gin.H{
			"total":                 acTotal,
			"passed":                acPassed,
			"failed":                acFailed,
			"completion_percentage": acCompletion,
		},
	})
}

func (h *ReportHandler) Burndown(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}
	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "项目不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	sprintID, ok := parseUintQuery(c, "sprint_id")
	if !ok {
		api.BadRequest(c, "sprint_id无效")
		return
	}

	sprint, err := h.sprintRepo.FindByID(sprintID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "冲刺不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}
	if sprint.ProjectID != projectID {
		api.BadRequest(c, "冲刺不属于当前项目")
		return
	}

	stories, err := h.storyRepo.ListBySprint(sprint.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	baseline := 0
	storyPoints := make(map[uint]int, len(stories))
	storyIDs := make([]uint, 0, len(stories))
	for _, s := range stories {
		p := storyPointValue(s)
		baseline += p
		storyPoints[s.ID] = p
		storyIDs = append(storyIDs, s.ID)
	}

	start := sprint.StartDate
	end := sprint.EndDate
	if end.Before(start) {
		end = start
	}

	today := time.Now()
	if today.Before(start) {
		today = start
	}
	if today.After(end) {
		today = end
	}
	todayEnd := endOfDay(today)

	logsByStory := map[uint][]model.ActivityLog{}
	if len(storyIDs) > 0 {
		var logs []model.ActivityLog
		if err := h.db.
			Where("entity_type = ? AND action = ? AND entity_id IN ? AND created_at <= ?", "story", "status_changed", storyIDs, todayEnd).
			Order("entity_id ASC, created_at DESC").
			Find(&logs).Error; err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		for _, log := range logs {
			logsByStory[log.EntityID] = append(logsByStory[log.EntityID], log)
		}
	}

	doneRangesByStory := make(map[uint][]burndownDoneRange, len(stories))
	for _, story := range stories {
		doneRangesByStory[story.ID] = buildDoneRanges(story, logsByStory[story.ID], todayEnd.Add(time.Nanosecond))
	}

	points := []gin.H{}
	for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
		dayEnd := endOfDay(d)
		remaining := baseline
		for _, s := range stories {
			if isDoneAt(dayEnd, doneRangesByStory[s.ID]) {
				remaining -= storyPoints[s.ID]
			}
		}
		if remaining < 0 {
			remaining = 0
		}
		points = append(points, gin.H{
			"date":             d.Format("2006-01-02"),
			"remaining_points": remaining,
		})
	}

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"sprint": gin.H{
			"id":         sprint.ID,
			"name":       sprint.Name,
			"start_date": sprint.StartDate,
			"end_date":   sprint.EndDate,
		},
		"baseline_points": baseline,
		"points":          points,
	})
}

func storyPointValue(story model.UserStory) int {
	if story.Points == nil || *story.Points <= 0 {
		return 1
	}
	return *story.Points
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

func parseStatusFromJSON(raw []byte) string {
	m, err := model.ParseJSONMap(raw)
	if err != nil {
		return ""
	}
	status, ok := m["status"].(string)
	if !ok {
		return ""
	}
	return status
}

func buildDoneRanges(story model.UserStory, logs []model.ActivityLog, horizon time.Time) []burndownDoneRange {
	if len(logs) == 0 {
		if story.Status != model.StoryStatusDone {
			return nil
		}
		start := story.UpdatedAt
		if start.After(horizon) {
			start = horizon
		}
		return []burndownDoneRange{{start: start, end: horizon}}
	}

	ranges := make([]burndownDoneRange, 0, 2)
	cursor := horizon
	status := story.Status

	for _, log := range logs {
		if status == model.StoryStatusDone && !log.CreatedAt.After(cursor) {
			ranges = append(ranges, burndownDoneRange{
				start: log.CreatedAt,
				end:   cursor,
			})
		}
		oldStatus := parseStatusFromJSON(log.OldValue)
		if oldStatus != "" {
			status = oldStatus
		}
		cursor = log.CreatedAt
	}

	if status == model.StoryStatusDone {
		ranges = append(ranges, burndownDoneRange{
			start: time.Time{},
			end:   cursor,
		})
	}

	return ranges
}

func isDoneAt(t time.Time, ranges []burndownDoneRange) bool {
	for _, r := range ranges {
		if !t.Before(r.start) && t.Before(r.end) {
			return true
		}
	}
	return false
}
