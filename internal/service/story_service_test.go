package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// logCapture is a test helper that intercepts slog records so assertions can
// verify that warnings/errors were emitted without coupling to the global logger.
type logCapture struct {
	records []slog.Record
	attrs   []slog.Attr
	groups  []string
}

func (c *logCapture) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (c *logCapture) Handle(_ context.Context, r slog.Record) error {
	c.records = append(c.records, r)
	return nil
}
func (c *logCapture) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *c
	cp.attrs = append(cp.attrs, attrs...)
	return &cp
}
func (c *logCapture) WithGroup(name string) slog.Handler {
	cp := *c
	cp.groups = append(cp.groups, name)
	return &cp
}

// findRecord returns the first record whose message equals msg and whose level
// is at least minLevel.
func (c *logCapture) findRecord(msg string, minLevel slog.Level) (slog.Record, bool) {
	for _, r := range c.records {
		if r.Level >= minLevel && r.Message == msg {
			return r, true
		}
	}
	return slog.Record{}, false
}

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
	db := setupStoryServiceTestDB(t)
	spy := &storyIndexSpy{
		shouldFail:   true,
		errorMessage: "embedding service unavailable",
	}
	svc := NewStoryServiceWithVector(db, nil, spy)

	capture := &logCapture{}
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(capture))
	defer slog.SetDefault(oldLogger)

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
	// 验证错误日志被记录
	rec, ok := capture.findRecord("failed to index story", slog.LevelWarn)
	require.True(t, ok, "expected warning log for index failure")
	var errVal slog.Value
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == "error" {
			errVal = a.Value
			return false
		}
		return true
	})
	require.Contains(t, errVal.String(), "embedding service unavailable")
}

func TestStoryServiceUpdateLogsIndexError(t *testing.T) {
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

	capture := &logCapture{}
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(capture))
	defer slog.SetDefault(oldLogger)

	newTitle := "统一登录"
	// 更新应该成功，即使索引失败
	changed, err := svc.Update(story, 1, UpdateStoryInput{Title: &newTitle})
	require.NoError(t, err)
	require.True(t, changed)
	// 验证错误日志被记录
	rec, ok := capture.findRecord("failed to index story", slog.LevelWarn)
	require.True(t, ok, "expected warning log for index failure")
	var errVal slog.Value
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == "error" {
			errVal = a.Value
			return false
		}
		return true
	})
	require.Contains(t, errVal.String(), "vector database connection lost")
}

func setupStoryServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:story_service_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserStory{}))
	return db
}
