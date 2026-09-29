package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedACStory 建一个带 1 条 AC 的故事，返回值拷贝 story 供测试模拟
// "调用方长期持有旧快照" 的并发场景。
func seedACStory(t *testing.T) (*gorm.DB, *StoryService, model.UserStory) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserStory{}, &model.ActivityLog{}))

	story := model.UserStory{
		ProjectID: 1, Title: "并发故事", StoryType: "feature", Status: model.StoryStatusBacklog,
		Priority: 1, CreatedBy: 1,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-1", Description: "已有标准", Status: model.ACStatusPending, Order: 1},
		}),
		Tags: []byte(`[]`), CodeReferences: []byte(`[]`),
	}
	require.NoError(t, db.Create(&story).Error)
	return db, NewStoryService(db, nil), story
}

func loadACs(t *testing.T, db *gorm.DB, storyID uint) ([]model.AcceptanceCriterion, model.UserStory) {
	t.Helper()
	var fresh model.UserStory
	require.NoError(t, db.First(&fresh, storyID).Error)
	criteria, err := model.ParseAcceptanceCriteria(fresh.AcceptanceCriteria)
	require.NoError(t, err)
	return criteria, fresh
}

// 两次 AddAC 之间故事被他人修改过（第二次调用仍持第一次之前的旧快照），
// 旧实现整行 Save 会用旧快照覆盖，丢掉第一次新增的 AC。
func TestACMutationDoesNotLoseConcurrentUpdates(t *testing.T) {
	db, svc, story := seedACStory(t)
	stale := story

	_, err := svc.AddAC(&stale, 1, "第一条新增", "", "", nil)
	require.NoError(t, err)
	_, err = svc.AddAC(&stale, 1, "第二条新增（仍持旧快照）", "", "", nil)
	require.NoError(t, err)

	criteria, _ := loadACs(t, db, story.ID)
	assert.Len(t, criteria, 3, "两次新增都必须保留，不得被旧快照覆盖")
}

// AC 变更只允许写 acceptance_criteria 列：并发改掉的状态等字段不得被回滚。
func TestACMutationPreservesConcurrentColumnChanges(t *testing.T) {
	db, svc, story := seedACStory(t)
	stale := story

	require.NoError(t, db.Model(&model.UserStory{}).Where("id = ?", story.ID).
		Updates(map[string]any{"status": model.StoryStatusTest, "version": gorm.Expr("version + 1")}).Error)

	_, err := svc.AddAC(&stale, 1, "状态变更之后新增", "", "", nil)
	require.NoError(t, err)

	criteria, fresh := loadACs(t, db, story.ID)
	assert.Equal(t, model.StoryStatusTest, fresh.Status, "并发改掉的状态不得被 AC 写入回滚")
	assert.Len(t, criteria, 2)
	assert.Equal(t, 2, fresh.Version, "乐观锁版本号应推进")
}

// 编辑必须作用于数据库里的最新 AC 列表：并发新增的 ac-2 不得被编辑操作吞掉。
func TestUpdateACAppliesToFreshState(t *testing.T) {
	db, svc, story := seedACStory(t)
	stale := story

	// 快照之后，另一人新增了 ac-2 并推进版本
	merged := []model.AcceptanceCriterion{
		{ID: "ac-1", Description: "已有标准", Status: model.ACStatusPending, Order: 1},
		{ID: "ac-2", Description: "并发新增", Status: model.ACStatusPending, Order: 2},
	}
	require.NoError(t, db.Model(&model.UserStory{}).Where("id = ?", story.ID).
		Updates(map[string]any{
			"acceptance_criteria": model.MarshalJSON(merged),
			"version":             gorm.Expr("version + 1"),
		}).Error)

	newDesc := "修改后的标准"
	got, err := svc.UpdateAC(&stale, 1, "ac-1", &newDesc, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "修改后的标准", got.Description)

	criteria, _ := loadACs(t, db, story.ID)
	assert.Len(t, criteria, 2, "并发新增的 ac-2 不得丢失")
	var edited string
	for _, ac := range criteria {
		if ac.ID == "ac-1" {
			edited = ac.Description
		}
	}
	assert.Equal(t, "修改后的标准", edited)
}
