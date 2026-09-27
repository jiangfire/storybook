import { describe, it, expect } from 'vitest';
import type { StoryBoardItem, User } from '../../../types/models';
import {
  collectAssignees,
  countBoardStories,
  emptyBoardFilters,
  filterBoardColumns,
  hasActiveFilters,
} from '../boardFilters';

const NOW = '2026-03-29T00:00:00Z';

function makeUser(id: number, email: string): User {
  return { id, email, role: 'developer', created_at: NOW };
}

function makeStory(overrides: Partial<StoryBoardItem> = {}): StoryBoardItem {
  return {
    id: 1,
    title: '支持手机号登录',
    story_type: 'feature',
    status: 'ready',
    priority: 2,
    ...overrides,
  };
}

const boardData: Record<string, StoryBoardItem[]> = {
  backlog: [
    makeStory({ id: 1, priority: 4, assigned_to: makeUser(9, 'dev@example.com') }),
    makeStory({ id: 2, title: 'Export 报表功能', priority: 1, sprint_id: 3 }),
  ],
  in_progress: [
    makeStory({ id: 3, status: 'in_progress', assigned_to: makeUser(10, 'dev2@example.com') }),
  ],
  done: [],
};

describe('boardFilters', () => {
  it('无激活条件时原样返回同一引用', () => {
    expect(filterBoardColumns(boardData, emptyBoardFilters)).toBe(boardData);
    expect(hasActiveFilters(emptyBoardFilters)).toBe(false);
  });

  it('按负责人过滤', () => {
    const result = filterBoardColumns(boardData, { ...emptyBoardFilters, assigneeId: 9 });
    expect(result.backlog.map((s) => s.id)).toEqual([1]);
    expect(result.in_progress).toEqual([]);
  });

  it('按优先级过滤', () => {
    const result = filterBoardColumns(boardData, { ...emptyBoardFilters, priority: 4 });
    expect(result.backlog.map((s) => s.id)).toEqual([1]);
  });

  it('按冲刺过滤', () => {
    const result = filterBoardColumns(boardData, { ...emptyBoardFilters, sprintId: 3 });
    expect(result.backlog.map((s) => s.id)).toEqual([2]);
  });

  it('关键词不区分大小写匹配标题', () => {
    const result = filterBoardColumns(boardData, { ...emptyBoardFilters, keyword: '报表' });
    expect(result.backlog.map((s) => s.id)).toEqual([2]);
    const upper = filterBoardColumns(boardData, { ...emptyBoardFilters, keyword: 'EXPORT' });
    expect(upper.backlog.map((s) => s.id)).toEqual([2]);
  });

  it('多条件取交集', () => {
    const result = filterBoardColumns(boardData, {
      ...emptyBoardFilters,
      assigneeId: 9,
      priority: 4,
      keyword: '登录',
    });
    expect(result.backlog.map((s) => s.id)).toEqual([1]);

    const none = filterBoardColumns(boardData, {
      ...emptyBoardFilters,
      assigneeId: 9,
      priority: 1,
    });
    expect(none.backlog).toEqual([]);
  });

  it('countBoardStories 汇总各列数量', () => {
    expect(countBoardStories(boardData)).toBe(3);
    const filtered = filterBoardColumns(boardData, { ...emptyBoardFilters, assigneeId: 9 });
    expect(countBoardStories(filtered)).toBe(1);
  });

  it('collectAssignees 去重并按邮箱排序', () => {
    const assignees = collectAssignees(boardData);
    expect(assignees).toEqual([
      { id: 9, email: 'dev@example.com' },
      { id: 10, email: 'dev2@example.com' },
    ]);
  });
});
