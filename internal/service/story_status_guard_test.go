package service

import (
	"testing"

	"github.com/jiangfire/storybook/internal/model"
	"github.com/stretchr/testify/require"
)

// pending（含被驳回）状态不允许通过状态接口流转，必须走审批/重提。
func TestStoryServiceUpdateStatusRejectsPendingStory(t *testing.T) {
	db := setupReviewNotifyDB(t)

	svc := NewStoryService(db, nil)
	story := createPendingStoryForNotify(t, svc, 1)

	err := svc.UpdateStatus(story, 1, model.StoryStatusBacklog, 0, nil)
	require.Error(t, err)

	story.ReviewStatus = model.ReviewStatusRejected
	require.NoError(t, db.Save(story).Error)
	err = svc.UpdateStatus(story, 1, model.StoryStatusReady, 0, nil)
	require.Error(t, err)
}

// 非 pending 故事的状态流转不受影响。
func TestStoryServiceUpdateStatusAllowsPostApprovalTransitions(t *testing.T) {
	db := setupReviewNotifyDB(t)

	svc := NewStoryService(db, nil)
	story := createPendingStoryForNotify(t, svc, 1)

	story.Status = model.StoryStatusBacklog
	story.ReviewStatus = model.ReviewStatusApproved
	require.NoError(t, db.Save(story).Error)

	require.NoError(t, svc.UpdateStatus(story, 1, model.StoryStatusReady, 1, nil))
	require.Equal(t, model.StoryStatusReady, story.Status)
}
