import Button from '../ui/Button';
import type { SprintSummary } from '../../types/api';

interface SprintPlannerProps {
  sprintId: string;
  sprints: SprintSummary[];
  isLoading: boolean;
  canPlan: boolean;
  isSubmitting: boolean;
  error: string;
  onSprintChange: (sprintId: string) => void;
  onUpdate: () => void;
  className?: string;
}

export const SprintPlanner = ({
  sprintId,
  sprints,
  isLoading,
  canPlan,
  isSubmitting,
  error,
  onSprintChange,
  onUpdate,
  className = '',
}: SprintPlannerProps) => {
  return (
    <div className={className}>
      <label className="block text-sm font-medium text-text mb-2">冲刺规划</label>
      <div className="flex flex-col gap-2 sm:flex-row">
        <select
          value={sprintId}
          onChange={(e) => onSprintChange(e.target.value)}
          disabled={isLoading || !canPlan}
          className="flex-1 rounded-lg border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary disabled:bg-secondary-50"
        >
          <option value="">不加入冲刺</option>
          {sprints.map((sprint) => (
            <option key={sprint.id} value={sprint.id}>
              {sprint.name}
            </option>
          ))}
        </select>
        <Button
          size="sm"
          variant="secondary"
          onClick={onUpdate}
          disabled={isLoading || !canPlan}
          isLoading={isSubmitting}
        >
          更新冲刺
        </Button>
      </div>
      {isLoading && <p className="mt-2 text-xs text-text-light">冲刺列表加载中...</p>}
      {!canPlan && <p className="mt-2 text-xs text-text-light">仅产品经理可规划冲刺</p>}
      {error && <p className="mt-2 text-xs text-danger">{error}</p>}
    </div>
  );
};
