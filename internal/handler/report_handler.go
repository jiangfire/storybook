package handler

import (
	"errors"
	"sort"
	"strings"
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
	db         *gorm.DB
	storyRepo  *repository.StoryRepository
	sprintRepo *repository.SprintRepository
	taskRepo   *repository.TaskRepository
	bugRepo    *repository.BugRepository
	activityRepo *repository.ActivityLogRepository
}

type burndownDoneRange struct {
	start time.Time
	end   time.Time
}

func NewReportHandler(db *gorm.DB) *ReportHandler {
	return &ReportHandler{
		db:           db,
		storyRepo:    repository.NewStoryRepository(db),
		sprintRepo:   repository.NewSprintRepository(db),
		taskRepo:     repository.NewTaskRepository(db),
		bugRepo:      repository.NewBugRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
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
		respondAccessError(c, err, "项目不存在")
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	statusBreakdown := map[string]int64{}
	for _, st := range []string{model.BugStatusOpen, model.BugStatusInProgress, model.BugStatusResolved, model.BugStatusClosed} {
		var n int64
		logging.LogIfErr(h.bugRepo.DB().Model(&model.BugReport{}).Where("project_id = ? AND status = ?", projectID, st).Count(&n).Error, "count bugs by status failed", "project_id", projectID, "status", st)
		statusBreakdown[st] = n
	}

	severityBreakdown := map[string]int64{}
	for _, sv := range []string{model.BugSeverityLow, model.BugSeverityMedium, model.BugSeverityHigh, model.BugSeverityCritical} {
		var n int64
		logging.LogIfErr(h.bugRepo.DB().Model(&model.BugReport{}).Where("project_id = ? AND severity = ?", projectID, sv).Count(&n).Error, "count bugs by severity failed", "project_id", projectID, "severity", sv)
		severityBreakdown[sv] = n
	}

	var stories []model.UserStory
	logging.LogIfErr(h.storyRepo.DB().Where("project_id = ?", projectID).Find(&stories).Error, "load project stories failed", "project_id", projectID)
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
	logging.LogIfErr(h.bugRepo.DB().Model(&model.BugReport{}).Where("project_id = ?", projectID).Count(&totalBugs).Error, "count project bugs failed", "project_id", projectID)

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
		respondAccessError(c, err, "项目不存在")
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
		if err := h.activityRepo.DB().
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

// parseReportWindow extracts ?from=YYYY-MM-DD&to=YYYY-MM-DD (also accepts RFC3339).
// Empty / invalid falls back to [now-defaultDays, now]. The end bound is rolled to
// 23:59:59.999... when only a date is supplied so the window is inclusive.
func parseReportWindow(c *gin.Context, defaultDays int) (time.Time, time.Time) {
	now := time.Now()
	to := endOfDay(now)
	from := now.AddDate(0, 0, -defaultDays)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, now.Location())

	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		if t, err := parseDateOrTime(raw); err == nil {
			from = t
		}
	}
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		if t, err := parseDateOrTime(raw); err == nil {
			// Pure date should be inclusive: roll to end of that day.
			if len(raw) <= 10 {
				to = endOfDay(t)
			} else {
				to = t
			}
		}
	}
	if to.Before(from) {
		to = endOfDay(from)
	}
	return from, to
}

// resolveStatusAt walks the story's status_changed history and returns the status
// the story held at `at`. Logs must be ordered ASC by created_at. OldValue carries
// the previous status, NewValue carries the post-transition status — we use
// NewValue at-or-before `at`, falling back to OldValue of the first later log (which
// reflects the status before that transition), and finally story.Status as the
// current state when no log applies.
func resolveStatusAt(story model.UserStory, logs []model.ActivityLog, at time.Time) string {
	// Default: status at "now" — set later if no log straddles `at`.
	current := story.Status
	if at.Before(story.CreatedAt) {
		return ""
	}

	var lastApplied string
	for _, log := range logs {
		if log.CreatedAt.After(at) {
			// First log strictly after `at`: its OldValue is the status held at `at`
			// (assuming logs are dense; falls through to `current` otherwise).
			if lastApplied == "" {
				if old := parseStatusFromJSON(log.OldValue); old != "" {
					return old
				}
			}
			break
		}
		if s := parseStatusFromJSON(log.NewValue); s != "" {
			lastApplied = s
		}
	}
	if lastApplied != "" {
		return lastApplied
	}
	return current
}

// firstTransitionAt finds the earliest status_changed log whose new_value.status
// equals targetStatus. Returns zero time when no such transition exists.
func firstTransitionAt(logs []model.ActivityLog, targetStatus string) time.Time {
	for _, log := range logs {
		if parseStatusFromJSON(log.NewValue) == targetStatus {
			return log.CreatedAt
		}
	}
	return time.Time{}
}

// CumulativeFlow renders day-by-day status distribution across the window.
// Each point is a histogram of how many stories sat in each status at end-of-day.
func (h *ReportHandler) CumulativeFlow(c *gin.Context) {
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	from, to := parseReportWindow(c, 30)

	var stories []model.UserStory
	if err := h.storyRepo.DB().Where("project_id = ? AND created_at <= ?", projectID, to).Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	storyIDs := make([]uint, 0, len(stories))
	for _, s := range stories {
		storyIDs = append(storyIDs, s.ID)
	}

	logsByStory := map[uint][]model.ActivityLog{}
	if len(storyIDs) > 0 {
		var logs []model.ActivityLog
		if err := h.activityRepo.DB().
			Where("entity_type = ? AND action = ? AND entity_id IN ? AND created_at <= ?", "story", "status_changed", storyIDs, to).
			Order("entity_id ASC, created_at ASC").
			Find(&logs).Error; err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		for _, log := range logs {
			logsByStory[log.EntityID] = append(logsByStory[log.EntityID], log)
		}
	}

	statuses := []string{
		model.StoryStatusPending,
		model.StoryStatusBacklog,
		model.StoryStatusReady,
		model.StoryStatusInProgress,
		model.StoryStatusTest,
		model.StoryStatusDone,
	}

	points := []gin.H{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dayEnd := endOfDay(d)
		if dayEnd.After(to) {
			dayEnd = to
		}
		counts := map[string]int{}
		for _, st := range statuses {
			counts[st] = 0
		}
		for _, s := range stories {
			if s.CreatedAt.After(dayEnd) {
				continue
			}
			status := resolveStatusAt(s, logsByStory[s.ID], dayEnd)
			if status == "" {
				continue
			}
			counts[status]++
		}
		points = append(points, gin.H{
			"date":     d.Format("2006-01-02"),
			"statuses": counts,
		})
	}

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"from":       from.Format("2006-01-02"),
		"to":         to.Format("2006-01-02"),
		"statuses":   statuses,
		"points":     points,
	})
}

// CycleTime averages the duration from first `in_progress` transition to first
// `done` transition across stories that completed within the window.
func (h *ReportHandler) CycleTime(c *gin.Context) {
	h.respondTimeMetric(c, "cycle_time", func(story model.UserStory, logs []model.ActivityLog) (time.Time, time.Time, bool) {
		startedAt := firstTransitionAt(logs, model.StoryStatusInProgress)
		doneAt := firstTransitionAt(logs, model.StoryStatusDone)
		if startedAt.IsZero() || doneAt.IsZero() || !doneAt.After(startedAt) {
			return time.Time{}, time.Time{}, false
		}
		return startedAt, doneAt, true
	})
}

// LeadTime averages the duration from story creation to first `done` transition.
func (h *ReportHandler) LeadTime(c *gin.Context) {
	h.respondTimeMetric(c, "lead_time", func(story model.UserStory, logs []model.ActivityLog) (time.Time, time.Time, bool) {
		doneAt := firstTransitionAt(logs, model.StoryStatusDone)
		if doneAt.IsZero() || !doneAt.After(story.CreatedAt) {
			return time.Time{}, time.Time{}, false
		}
		return story.CreatedAt, doneAt, true
	})
}

// respondTimeMetric is the shared aggregator behind CycleTime and LeadTime: load
// stories completed in [from, to], compute per-story interval, average, respond.
func (h *ReportHandler) respondTimeMetric(c *gin.Context, label string, extract func(model.UserStory, []model.ActivityLog) (time.Time, time.Time, bool)) {
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	from, to := parseReportWindow(c, 90)

	var stories []model.UserStory
	if err := h.storyRepo.DB().Where("project_id = ? AND status = ?", projectID, model.StoryStatusDone).Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	storyIDs := make([]uint, 0, len(stories))
	for _, s := range stories {
		storyIDs = append(storyIDs, s.ID)
	}

	logsByStory := map[uint][]model.ActivityLog{}
	if len(storyIDs) > 0 {
		var logs []model.ActivityLog
		if err := h.activityRepo.DB().
			Where("entity_type = ? AND action = ? AND entity_id IN ?", "story", "status_changed", storyIDs).
			Order("entity_id ASC, created_at ASC").
			Find(&logs).Error; err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		for _, log := range logs {
			logsByStory[log.EntityID] = append(logsByStory[log.EntityID], log)
		}
	}

	perStory := make([]gin.H, 0, len(stories))
	var totalHours float64
	var samples int
	for _, s := range stories {
		startAt, endAt, ok := extract(s, logsByStory[s.ID])
		if !ok {
			continue
		}
		if endAt.Before(from) || endAt.After(to) {
			continue
		}
		hours := endAt.Sub(startAt).Hours()
		totalHours += hours
		samples++
		perStory = append(perStory, gin.H{
			"story_id": s.ID,
			"title":    s.Title,
			"hours":    hours,
			"days":     hours / 24.0,
		})
	}

	sort.Slice(perStory, func(i, j int) bool {
		hi, _ := perStory[i]["hours"].(float64)
		hj, _ := perStory[j]["hours"].(float64)
		return hi > hj
	})

	avgHours := 0.0
	if samples > 0 {
		avgHours = totalHours / float64(samples)
	}

	api.Success(c, "success", gin.H{
		"project_id":  projectID,
		"metric":      label,
		"from":        from.Format("2006-01-02"),
		"to":          to.Format("2006-01-02"),
		"sample_size": samples,
		"average_hours": avgHours,
		"average_days":  avgHours / 24.0,
		"per_story":     perStory,
	})
}

// Throughput counts stories whose first `done` transition lands in each interval
// of the window. interval=week (default) or interval=day.
func (h *ReportHandler) Throughput(c *gin.Context) {
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	interval := strings.ToLower(strings.TrimSpace(c.Query("interval")))
	if interval != "day" {
		interval = "week"
	}
	defaultDays := 84 // 12 weeks
	if interval == "day" {
		defaultDays = 30
	}
	from, to := parseReportWindow(c, defaultDays)

	var stories []model.UserStory
	if err := h.storyRepo.DB().Where("project_id = ? AND status = ?", projectID, model.StoryStatusDone).Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	storyIDs := make([]uint, 0, len(stories))
	for _, s := range stories {
		storyIDs = append(storyIDs, s.ID)
	}

	logsByStory := map[uint][]model.ActivityLog{}
	if len(storyIDs) > 0 {
		var logs []model.ActivityLog
		if err := h.activityRepo.DB().
			Where("entity_type = ? AND action = ? AND entity_id IN ?", "story", "status_changed", storyIDs).
			Order("entity_id ASC, created_at ASC").
			Find(&logs).Error; err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		for _, log := range logs {
			logsByStory[log.EntityID] = append(logsByStory[log.EntityID], log)
		}
	}

	completionByStory := make(map[uint]time.Time, len(stories))
	for _, s := range stories {
		if doneAt := firstTransitionAt(logsByStory[s.ID], model.StoryStatusDone); !doneAt.IsZero() {
			completionByStory[s.ID] = doneAt
		}
	}

	// Build bucket boundaries.
	type bucket struct {
		start time.Time
		end   time.Time
	}
	var buckets []bucket
	if interval == "day" {
		for d := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()); !d.After(to); d = d.AddDate(0, 0, 1) {
			buckets = append(buckets, bucket{start: d, end: endOfDay(d)})
		}
	} else {
		// Align to ISO week (Monday). Walk by week.
		start := from
		offset := int(start.Weekday()) - 1
		if offset < 0 {
			offset = 6
		}
		start = start.AddDate(0, 0, -offset)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		for d := start; !d.After(to); d = d.AddDate(0, 0, 7) {
			end := endOfDay(d.AddDate(0, 0, 6))
			if end.After(to) {
				end = to
			}
			buckets = append(buckets, bucket{start: d, end: end})
		}
	}

	points := make([]gin.H, 0, len(buckets))
	totalCompleted := 0
	for _, b := range buckets {
		count := 0
		for _, doneAt := range completionByStory {
			if !doneAt.Before(b.start) && !doneAt.After(b.end) {
				count++
			}
		}
		totalCompleted += count
		points = append(points, gin.H{
			"period_start":    b.start.Format("2006-01-02"),
			"period_end":      b.end.Format("2006-01-02"),
			"completed_count": count,
		})
	}

	api.Success(c, "success", gin.H{
		"project_id":      projectID,
		"interval":        interval,
		"from":            from.Format("2006-01-02"),
		"to":              to.Format("2006-01-02"),
		"total_completed": totalCompleted,
		"points":          points,
	})
}
