import { formatStoryStatus } from '../../../utils/formatters';
import type { StatusBreakdown } from '../../../types/models';

const orderedStatuses: Array<keyof StatusBreakdown> = [
  'pending',
  'backlog',
  'ready',
  'in_progress',
  'test',
  'done',
];

export function ProjectStatusSection({ statusBreakdown }: { statusBreakdown: StatusBreakdown }) {
  return (
    <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-text">状态分布</h2>
        <p className="mt-1 text-sm text-text-light">快速看当前故事主要积压在哪个阶段。</p>
      </div>
      <div className="space-y-2">
        {orderedStatuses.map((status) => (
          <div
            key={status}
            className="section-block flex items-center justify-between rounded-[1.1rem] px-3 py-2.5"
          >
            <span className="text-text-light">{formatStoryStatus(status)}</span>
            <span className="font-medium text-text">{statusBreakdown[status] || 0}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
