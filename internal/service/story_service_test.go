package service

import (
	"context"
	"errors"
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type storyIndexSpy struct {
	indexed      []uint
	shouldFail   bool
	errorMessage string
}

func (s *storyIndexSpy) SearchSimilarStories(ctx context.Context, query string, projectIDs []uint, limit int) ([]SimilarStory, error) {
	return nil, nil
}

func (s *storyIndexSpy) IndexStory(ctx context.Context, story *model.UserStory) error {
	if s.shouldFail {
		return errors.New(s.errorMessage)
	}
	if story != nil {
		s.indexed = append(s.indexed, story.ID)
	}
	return nil
}

func (s *storyIndexSpy) BatchIndexStories(ctx context.Context, stories []model.UserStory) error {
	return nil
}

func (s *storyIndexSpy) PrepareStoryContent(story *model.UserStory) string {
	return ""
}

func TestStoryServiceCreateIndexesStoryWhenVectorEnabled(t *testing.T) {
	db := setupStoryServiceTestDB(t)
	spy := &storyIndexSpy{}
	svc := NewStoryServiceWithVector(db, nil, spy)

	story, err := svc.Create(CreateStoryInput{
		ProjectID:          1,
		UserID:             1,
		Title:              "用户登录",
		Description:        "支持用户名密码登录",
		StoryType:          model.StoryTypeFeature,
		Priority:           1,
		AcceptanceCriteria: []model.AcceptanceCriterion{{ID: "ac-1", Description: "可登录", Order: 1}},
		Tags:               []string{"auth"},
		Position:           1,
	})
	require.NoError(t, err)
	require.NotNil(t, story)
	require.Equal(t, []uint{story.ID}, spy.indexed)
}

func TestStoryServiceUpdateReindexesOnContentChange(t *testing.T) {
	db := setupStoryServiceTestDB(t)
	spy := &storyIndexSpy{}
	svc := NewStoryServiceWithVector(db, nil, spy)

	story, err := svc.Create(CreateStoryInput{
		ProjectID:          1,
		UserID:             1,
		Title:              "用户登录",
		Description:        "支持用户名密码登录",
		StoryType:          model.StoryTypeFeature,
		Priority:           1,
		AcceptanceCriteria: []model.AcceptanceCriterion{{ID: "ac-1", Description: "可登录", Order: 1}},
		Tags:               []string{"auth"},
		Position:           1,
	})
	require.NoError(t, err)
	spy.indexed = nil

	newTitle := "统一登录"
	changed, err := svc.Update(story, 1, UpdateStoryInput{Title: &newTitle})
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, []uint{story.ID}, spy.indexed)
}

func TestStoryServiceCreateLogsIndexError(t *testing.T) {
	// TODO: 需要实现日志捕获机制来验证错误被记录
	// 这是一个占位测试，记录当前期望
	db := setupStoryServiceTestDB(t)
	spy := &storyIndexSpy{
		shouldFail:   true,
		errorMessage: "embedding service unavailable",
	}
	svc := NewStoryServiceWithVector(db, nil, spy)

	// 故事创建应该成功，即使索引失败
	story, err := svc.Create(CreateStoryInput{
		ProjectID:          1,
		UserID:             1,
		Title:              "用户登录",
		Description:        "支持用户名密码登录",
		StoryType:          model.StoryTypeFeature,
		Priority:           1,
		AcceptanceCriteria: []model.AcceptanceCriterion{{ID: "ac-1", Description: "可登录", Order: 1}},
		Tags:               []string{"auth"},
		Position:           1,
	})
	require.NoError(t, err)
	require.NotNil(t, story)
	// 索引应该失败了
	require.Empty(t, spy.indexed)
	// TODO: 验证错误日志被记录
}

func TestStoryServiceUpdateLogsIndexError(t *testing.T) {
	// 测试更新时索引失败的情况
	db := setupStoryServiceTestDB(t)
	spy := &storyIndexSpy{}
	svc := NewStoryServiceWithVector(db, nil, spy)

	story, err := svc.Create(CreateStoryInput{
		ProjectID:          1,
		UserID:             1,
		Title:              "用户登录",
		Description:        "支持用户名密码登录",
		StoryType:          model.StoryTypeFeature,
		Priority:           1,
		AcceptanceCriteria: []model.AcceptanceCriterion{{ID: "ac-1", Description: "可登录", Order: 1}},
		Tags:               []string{"auth"},
		Position:           1,
	})
	require.NoError(t, err)

	// 设置spy在更新时失败
	spy.shouldFail = true
	spy.errorMessage = "vector database connection lost"

	newTitle := "统一登录"
	// 更新应该成功，即使索引失败
	changed, err := svc.Update(story, 1, UpdateStoryInput{Title: &newTitle})
	require.NoError(t, err)
	require.True(t, changed)
	// TODO: 验证错误日志被记录
}

func setupStoryServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:story_service_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserStory{}))
	return db
}
