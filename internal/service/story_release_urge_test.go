package service

import (
	"testing"
	"time"

	"github.com/jiangfire/storybook/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func releaseTestStory(t *testing.T, db *gorm.DB, status string) *model.UserStory {
	t.Helper()
	assignee := uint(2)
	story := model.UserStory{
		ProjectID:          1,
		Title:              "释放回退",
		StoryType:          model.StoryTypeFeature,
		Status:             status,
		ReviewStatus:       model.ReviewStatusApproved,
		CreatedBy:          1,
		AssignedTo:         &assignee,
		AcceptanceCriteria: model.MarshalJSON([]any{}),
		Tags:               model.MarshalJSON([]any{}),
		CodeReferences:     model.MarshalJSON([]any{}),
	}
	require.NoError(t, db.Create(&story).Error)
	return &story
}

func TestStoryServiceReleaseStepsBackOneStatus(t *testing.T) {
	cases := []struct {
		from string
		want string
	}{
		{model.StoryStatusTest, model.StoryStatusInProgress},
		{model.StoryStatusInProgress, model.StoryStatusReady},
		{model.StoryStatusBacklog, model.StoryStatusBacklog},
	}

	for _, tc := range cases {
		db := setupReviewNotifyDB(t)
		svc := NewStoryService(db, nil)
		story := releaseTestStory(t, db, tc.from)

		require.NoError(t, svc.Release(story, 2, "developer"))

		var reloaded model.UserStory
		require.NoError(t, db.First(&reloaded, story.ID).Error)
		require.Nil(t, reloaded.AssignedTo)
		require.Equal(t, tc.want, reloaded.Status, "release from %s should land on %s", tc.from, tc.want)
	}
}

func TestStoryServiceUrgeReviewCooldownBlocksRepeatedCalls(t *testing.T) {
	db := setupReviewNotifyDB(t)

	require.NoError(t, db.Create(&model.User{ID: 2, Username: "tl-cooldown", Email: "tl@example.com", Role: model.RoleTechLead}).Error)
	require.NoError(t, db.Create(&model.ProjectTechLead{ProjectID: 1, UserID: 2}).Error)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)
	notifier.calls = nil

	require.NoError(t, svc.UrgeReview(story, 1))
	err := svc.UrgeReview(story, 1)
	require.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Len(t, validationErr.Issues, 1)
	require.Contains(t, validationErr.Issues[0].Message, "5 分钟")
	require.Len(t, notifier.calls, 1)
}

func TestStoryServiceUrgeReviewCooldownAllowsAfterWindow(t *testing.T) {
	db := setupReviewNotifyDB(t)

	require.NoError(t, db.Create(&model.User{ID: 2, Username: "tl-window", Email: "tl@example.com", Role: model.RoleTechLead}).Error)
	require.NoError(t, db.Create(&model.ProjectTechLead{ProjectID: 1, UserID: 2}).Error)

	notifier := &recordingNotifier{}
	svc := NewStoryService(db, nil).WithNotifier(notifier)

	story := createPendingStoryForNotify(t, svc, 1)

	// 预置一条 6 分钟前的催审日志，冷却窗口已过。
	require.NoError(t, db.Create(&model.ActivityLog{
		EntityType: "story",
		EntityID:   story.ID,
		Action:     "review_urged",
		UserID:     1,
		ProjectID:  &story.ProjectID,
		CreatedAt:  time.Now().Add(-6 * time.Minute),
	}).Error)
	notifier.calls = nil

	require.NoError(t, svc.UrgeReview(story, 1))
	require.Len(t, notifier.calls, 1)
}
