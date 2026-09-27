import type { StoryBoardItem } from '../../types/models';

export interface BoardFilters {
  assigneeId: number | null;
  priority: number | null;
  sprintId: number | null;
  keyword: string;
}

export const emptyBoardFilters: BoardFilters = {
  assigneeId: null,
  priority: null,
  sprintId: null,
  keyword: '',
};

export function hasActiveFilters(filters: BoardFilters): boolean {
  return (
    filters.assigneeId !== null ||
    filters.priority !== null ||
    filters.sprintId !== null ||
    filters.keyword.trim() !== ''
  );
}

function matchesFilters(story: StoryBoardItem, filters: BoardFilters): boolean {
  if (filters.assigneeId !== null && story.assigned_to?.id !== filters.assigneeId) {
    return false;
  }
  if (filters.priority !== null && story.priority !== filters.priority) {
    return false;
  }
  if (filters.sprintId !== null && story.sprint_id !== filters.sprintId) {
    return false;
  }
  const keyword = filters.keyword.trim().toLowerCase();
  if (keyword && !story.title.toLowerCase().includes(keyword)) {
    return false;
  }
  return true;
}

/**
 * 按筛选条件过滤各列故事；无激活条件时原样返回，避免无谓复制。
 * 排序/拖拽逻辑仍基于完整数据（KanbanBoard 内部），这里只影响展示。
 */
export function filterBoardColumns(
  data: Record<string, StoryBoardItem[]>,
  filters: BoardFilters
): Record<string, StoryBoardItem[]> {
  if (!hasActiveFilters(filters)) {
    return data;
  }
  const result: Record<string, StoryBoardItem[]> = {};
  for (const [status, stories] of Object.entries(data)) {
    result[status] = stories.filter((story) => matchesFilters(story, filters));
  }
  return result;
}

export function countBoardStories(data: Record<string, StoryBoardItem[]>): number {
  return Object.values(data).reduce((sum, stories) => sum + stories.length, 0);
}

/** 从看板数据中收集去重后的负责人选项（按邮箱排序）。 */
export function collectAssignees(
  data: Record<string, StoryBoardItem[]>
): Array<{ id: number; email: string }> {
  const seen = new Map<number, string>();
  for (const stories of Object.values(data)) {
    for (const story of stories) {
      const assignee = story.assigned_to;
      if (assignee && !seen.has(assignee.id)) {
        seen.set(assignee.id, assignee.email);
      }
    }
  }
  return [...seen.entries()]
    .map(([id, email]) => ({ id, email }))
    .sort((a, b) => a.email.localeCompare(b.email));
}
