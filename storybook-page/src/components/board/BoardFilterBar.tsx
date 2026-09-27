import Button from '../ui/Button';
import { formatPriority } from '../../utils/formatters';
import type { BoardFilters } from './boardFilters';
import type { SprintSummary } from '../../types/api';

interface BoardFilterBarProps {
  assignees: Array<{ id: number; email: string }>;
  sprints: SprintSummary[];
  filters: BoardFilters;
  onChange: (filters: BoardFilters) => void;
  totalCount: number;
  visibleCount: number;
}

const PRIORITY_OPTIONS = [4, 3, 2, 1, 0];

export function BoardFilterBar({
  assignees,
  sprints,
  filters,
  onChange,
  totalCount,
  visibleCount,
}: BoardFilterBarProps) {
  const active =
    filters.assigneeId !== null ||
    filters.priority !== null ||
    filters.sprintId !== null ||
    filters.keyword.trim() !== '';

  return (
    <div className="mb-4 flex flex-col gap-3 rounded-2xl border border-border bg-white p-3 lg:flex-row lg:items-center lg:flex-wrap">
      <div className="flex items-center gap-2">
        <label htmlFor="board-filter-assignee" className="whitespace-nowrap text-xs text-text-light">
          负责人
        </label>
        <select
          id="board-filter-assignee"
          value={filters.assigneeId ?? ''}
          onChange={(e) => onChange({ ...filters, assigneeId: e.target.value ? Number(e.target.value) : null })}
          className="field-control w-40"
        >
          <option value="">全部</option>
          {assignees.map((assignee) => (
            <option key={assignee.id} value={assignee.id}>
              {assignee.email}
            </option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-2">
        <label htmlFor="board-filter-priority" className="whitespace-nowrap text-xs text-text-light">
          优先级
        </label>
        <select
          id="board-filter-priority"
          value={filters.priority ?? ''}
          onChange={(e) => onChange({ ...filters, priority: e.target.value ? Number(e.target.value) : null })}
          className="field-control w-28"
        >
          <option value="">全部</option>
          {PRIORITY_OPTIONS.map((priority) => (
            <option key={priority} value={priority}>
              {formatPriority(priority)}
            </option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-2">
        <label htmlFor="board-filter-sprint" className="whitespace-nowrap text-xs text-text-light">
          冲刺
        </label>
        <select
          id="board-filter-sprint"
          value={filters.sprintId ?? ''}
          onChange={(e) => onChange({ ...filters, sprintId: e.target.value ? Number(e.target.value) : null })}
          className="field-control w-40"
        >
          <option value="">全部</option>
          {sprints.map((sprint) => (
            <option key={sprint.id} value={sprint.id}>
              {sprint.name}
            </option>
          ))}
        </select>
      </div>

      <div className="flex min-w-0 flex-1 items-center gap-2">
        <label htmlFor="board-filter-keyword" className="whitespace-nowrap text-xs text-text-light">
          关键词
        </label>
        <input
          id="board-filter-keyword"
          type="text"
          value={filters.keyword}
          onChange={(e) => onChange({ ...filters, keyword: e.target.value })}
          placeholder="按标题搜索"
          className="field-control w-full lg:max-w-xs"
        />
      </div>

      <div className="flex items-center justify-between gap-3 lg:justify-end">
        <span className="whitespace-nowrap text-xs text-text-light">
          {active ? `显示 ${visibleCount}/${totalCount}` : `共 ${totalCount} 个故事`}
        </span>
        {active && (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => onChange({ assigneeId: null, priority: null, sprintId: null, keyword: '' })}
          >
            清除筛选
          </Button>
        )}
      </div>
    </div>
  );
}

export default BoardFilterBar;
