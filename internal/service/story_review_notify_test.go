package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type notifyCall struct {
	userIDs []uint
	event   NotificationEvent
}

type recordingNotifier struct {
	calls []notifyCall
}

func (n *recordingNotifier) Notify(ctx context.Context, userID uint, ev NotificationEvent) {
	n.NotifyMany(ctx, []uint{userID}, ev)
}

func (n *recordingNotifier) NotifyMany(_ context.Context, userIDs []uint, ev NotificationEvent) {
	n.calls = append(n.calls, notifyCall{userIDs: append([]uint(nil), userIDs...), event: ev})
}

func (n *recordingNotifier) NotifyProjectMembers(context.Context, uint, NotificationEvent) {}

func setupReviewNotifyDB(t *testing.T) *gorm.DB {
	t.Helper()

	// 每个测试独立的内存库，避免 shared-cache 数据互相泄漏
	dsn := fmt.Sprintf("file:review_notify_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserStory{}, &model.User{}, &model.Project{}, &model.ProjectMember{}, &model.ProjectTechLead{}, &model.ActivityLog{}))
	return db
}

func createPendingStoryForNotify(t *testing.T, svc *StoryService, creatorID uint) *model.UserStory {
	t.Helper()

	story, err := svc.Create(CreateStoryInput{
		ProjectID: 1,
		UserID:    creatorID,
		Title:     "手机号登录",
		StoryType: model.StoryTypeFeature,
		Priority:  2,
	})
	require.NoError(t, err)
	return story
}

func TestStoryServiceCreateNotifiesTechLeadsAndAdmins(t *testing.T) {
	db := setupReviewNotifyDB(t)

	// 项目 1 的技术负责人是用户 2；平台管理员是用户 3；创建者是用户 1。
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "tl-2", Email: "tl@example.com", Role: model.RoleTechLead}).Error)
	require.NoError(t, db.Create(&model.User{ID: 3, Username: "admin-3", Email: "admin@example.com", Role: model.RoleAdmin}).Error)
	require.NoError(t, db.Create(&model.ProjectTechLead{ProjectID: 1, UserID: 2}).Error)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)

	require.Len(t, notifier.calls, 1)
	call := notifier.calls[0]
	require.Equal(t, model.NotificationStoryReviewRequested, call.event.Type)
	require.Equal(t, story.ID, call.event.EntityID)
	require.ElementsMatch(t, []uint{2, 3}, call.userIDs)
}

func TestStoryServiceCreateSkipsNotifyWhenNoReviewers(t *testing.T) {
	db := setupReviewNotifyDB(t)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	createPendingStoryForNotify(t, svc, 1)

	require.Empty(t, notifier.calls)
}

func TestStoryServiceCreateExcludesCreatorFromReviewers(t *testing.T) {
	db := setupReviewNotifyDB(t)

	// 创建者本人是管理员：不应给自己发审批通知。
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "pm-admin-1", Email: "pm-admin@example.com", Role: model.RoleAdmin}).Error)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	createPendingStoryForNotify(t, svc, 1)

	require.Empty(t, notifier.calls)
}

func TestStoryServiceResubmitReviewResetsStatusAndNotifies(t *testing.T) {
	db := setupReviewNotifyDB(t)

	require.NoError(t, db.Create(&model.User{ID: 2, Username: "tl-resubmit", Email: "tl@example.com", Role: model.RoleTechLead}).Error)
	require.NoError(t, db.Create(&model.ProjectTechLead{ProjectID: 1, UserID: 2}).Error)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)
	notifier.calls = nil

	// 模拟审批驳回
	story.ReviewStatus = model.ReviewStatusRejected
	story.ReviewComment = "验收标准不可验证"
	story.ReviewedBy = &[]uint{2}[0]
	require.NoError(t, db.Save(story).Error)
	notifier.calls = nil

	require.NoError(t, svc.ResubmitReview(story, 1))

	require.Equal(t, model.ReviewStatusPending, story.ReviewStatus)
	require.Len(t, notifier.calls, 1)
	call := notifier.calls[0]
	require.Equal(t, model.NotificationStoryReviewResubmitted, call.event.Type)
	require.Equal(t, "故事已重新提交审批", call.event.Title)
	require.ElementsMatch(t, []uint{2}, call.userIDs)
}

func TestStoryServiceResubmitReviewRejectsInvalidState(t *testing.T) {
	db := setupReviewNotifyDB(t)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)

	// 审批通过进入 backlog 后不允许“重提”
	story.Status = model.StoryStatusBacklog
	story.ReviewStatus = model.ReviewStatusApproved
	require.NoError(t, db.Save(story).Error)

	err := svc.ResubmitReview(story, 1)
	require.Error(t, err)
	require.Empty(t, notifier.calls)
}

func TestStoryServiceUrgeReviewNotifiesReviewers(t *testing.T) {
	db := setupReviewNotifyDB(t)

	require.NoError(t, db.Create(&model.User{ID: 2, Username: "tl-urge", Email: "tl@example.com", Role: model.RoleTechLead}).Error)
	require.NoError(t, db.Create(&model.ProjectTechLead{ProjectID: 1, UserID: 2}).Error)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)
	notifier.calls = nil

	require.NoError(t, svc.UrgeReview(story, 1))

	// 催审不改变故事状态
	require.Equal(t, model.StoryStatusPending, story.Status)
	require.Len(t, notifier.calls, 1)
	call := notifier.calls[0]
	require.Equal(t, model.NotificationStoryReviewUrged, call.event.Type)
	require.Equal(t, "有故事被催促审批", call.event.Title)
	require.ElementsMatch(t, []uint{2}, call.userIDs)
}

func TestStoryServiceUrgeReviewRejectsNonPendingStory(t *testing.T) {
	db := setupReviewNotifyDB(t)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)

	story.Status = model.StoryStatusInProgress
	require.NoError(t, db.Save(story).Error)

	err := svc.UrgeReview(story, 1)
	require.Error(t, err)
	require.Empty(t, notifier.calls)
}
