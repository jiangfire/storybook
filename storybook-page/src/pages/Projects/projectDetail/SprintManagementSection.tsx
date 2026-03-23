import Button from '../../../components/ui/Button';
import { formatSprintStatus } from '../../../utils/formatters';
import type { SprintSummary } from '../../../types/api';
import { getNextSprintAction, getSprintStatusClass } from './sprintHelpers';

interface SprintManagementSectionProps {
  sprintError: string;
  sprints: SprintSummary[];
  statusUpdatingSprintID: number | null;
  onCreateSprint: () => void;
  onSelectSprint: (sprintID: number) => void;
  onUpdateSprintStatus: (sprint: SprintSummary) => Promise<void>;
}

export function SprintManagementSection({
  sprintError,
  sprints,
  statusUpdatingSprintID,
  onCreateSprint,
  onSelectSprint,
  onUpdateSprintStatus,
}: SprintManagementSectionProps) {
  return (
    <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">冲刺管理</h2>
          <p className="mt-1 text-sm text-text-light">统一查看每个冲刺的周期、完成量和下一步动作。</p>
        </div>
        <Button size="sm" onClick={onCreateSprint}>
          + 新建冲刺
        </Button>
      </div>

      {sprintError && <div className="mb-3 state-panel state-panel-error">{sprintError}</div>}
      {sprints.length === 0 ? (
        <div className="state-panel state-panel-empty">暂无冲刺，先创建一个冲刺</div>
      ) : (
        <div className="space-y-2">
          {sprints.map((sprint) => {
            const action = getNextSprintAction(sprint.status);

            return (
              <div
                key={sprint.id}
                className="section-block flex flex-col gap-3 rounded-[1.2rem] px-4 py-3 md:flex-row md:items-center md:justify-between"
              >
                <div className="min-w-0">
                  <div className="mb-1 flex items-center gap-2">
                    <span className="truncate font-medium text-text">{sprint.name}</span>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs ${getSprintStatusClass(sprint.status)}`}
                    >
                      {formatSprintStatus(sprint.status)}
                    </span>
                  </div>
                  <div className="text-xs text-text-light">
                    {new Date(sprint.start_date).toLocaleDateString()} -{' '}
                    {new Date(sprint.end_date).toLocaleDateString()} · {sprint.done_stories}/
                    {sprint.total_stories} 故事完成
                  </div>
                </div>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Button size="sm" variant="secondary" onClick={() => onSelectSprint(sprint.id)}>
                    查看燃尽图
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => void onUpdateSprintStatus(sprint)}
                    isLoading={statusUpdatingSprintID === sprint.id}
                  >
                    {action.label}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
