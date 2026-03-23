import { formatStoryStatus } from '../../../utils/formatters';
import type { StatusBreakdown } from '../../../types/models';

export function ProjectStatusSection({ statusBreakdown }: { statusBreakdown: StatusBreakdown }) {
  return (
    <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-text">状态分布</h2>
        <p className="mt-1 text-sm text-text-light">快速看当前故事主要积压在哪个阶段。</p>
      </div>
      <div className="space-y-2">
        {Object.entries(statusBreakdown).map(([status, count]) => (
          <div
            key={status}
            className="section-block flex items-center justify-between rounded-[1.1rem] px-3 py-2.5"
          >
            <span className="text-text-light">{formatStoryStatus(status)}</span>
            <span className="font-medium text-text">{count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
