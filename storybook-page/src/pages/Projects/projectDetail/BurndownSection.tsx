import Button from '../../../components/ui/Button';
import type { BurndownReport, SprintSummary } from '../../../types/api';

interface BurndownSectionProps {
  projectID: number;
  sprintError: string;
  sprints: SprintSummary[];
  selectedSprintID: number | null;
  isBurndownLoading: boolean;
  burndownError: string;
  burndown: BurndownReport | null;
  onSelectSprint: (sprintID: number | null) => void;
  onRefresh: (projectID: number, sprintID: number) => void;
}

export function BurndownSection({
  projectID,
  sprintError,
  sprints,
  selectedSprintID,
  isBurndownLoading,
  burndownError,
  burndown,
  onSelectSprint,
  onRefresh,
}: BurndownSectionProps) {
  return (
    <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">燃尽图</h2>
          <p className="mt-1 text-sm text-text-light">把理想线和实际线放在一起看，快速判断冲刺节奏是否健康。</p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <span className="text-sm text-text-light">冲刺</span>
          <select
            value={selectedSprintID || ''}
            onChange={(event) => onSelectSprint(event.target.value ? Number(event.target.value) : null)}
            className="field-control"
            disabled={sprints.length === 0}
          >
            {sprints.length === 0 && <option value="">暂无冲刺</option>}
            {sprints.map((sprint) => (
              <option key={sprint.id} value={sprint.id}>
                {sprint.name}
              </option>
            ))}
          </select>
          <Button
            size="sm"
            variant="secondary"
            disabled={!selectedSprintID || isBurndownLoading}
            onClick={() => {
              if (selectedSprintID) {
                onRefresh(projectID, selectedSprintID);
              }
            }}
          >
            刷新
          </Button>
        </div>
      </div>

      {sprintError && <div className="mb-3 state-panel state-panel-error">{sprintError}</div>}
      {isBurndownLoading && <div className="state-panel state-panel-loading">燃尽图加载中...</div>}
      {!isBurndownLoading && burndownError && <div className="state-panel state-panel-error">{burndownError}</div>}
      {!isBurndownLoading && !burndownError && burndown && <BurndownChart report={burndown} />}
      {!isBurndownLoading && !burndown && !burndownError && (
        <div className="state-panel state-panel-empty">请选择冲刺查看燃尽图</div>
      )}
    </div>
  );
}

function BurndownChart({ report }: { report: BurndownReport }) {
  const points = report.points || [];
  if (points.length === 0 || report.baseline_points <= 0) {
    return (
      <div className="state-panel state-panel-empty space-y-2">
        <div>当前冲刺暂无可燃尽的数据</div>
        <div>请先把故事规划到该冲刺，并设置故事点；完成故事后实际线才会下降。</div>
      </div>
    );
  }

  const width = 900;
  const height = 280;
  const paddingX = 40;
  const paddingY = 24;
  const plotWidth = width - paddingX * 2;
  const plotHeight = height - paddingY * 2;
  const maxY = Math.max(report.baseline_points, ...points.map((point) => point.remaining_points), 1);
  const xDivisor = Math.max(points.length - 1, 1);
  const dayMS = 24 * 60 * 60 * 1000;

  const parseDateOnly = (raw: string) => {
    const datePart = raw.slice(0, 10);
    const [year, month, day] = datePart.split('-').map((value) => Number(value));
    return new Date(year, month - 1, day);
  };

  const sprintStart = parseDateOnly(report.sprint.start_date);
  const sprintEnd = parseDateOnly(report.sprint.end_date);
  const totalSprintDays = Math.max(
    Math.round((sprintEnd.getTime() - sprintStart.getTime()) / dayMS),
    1
  );

  const toX = (index: number) => paddingX + (plotWidth * index) / xDivisor;
  const toY = (value: number) => paddingY + plotHeight - (plotHeight * value) / maxY;
  const getIdealValue = (date: string) => {
    const offsetDays = Math.min(
      Math.max(Math.round((parseDateOnly(date).getTime() - sprintStart.getTime()) / dayMS), 0),
      totalSprintDays
    );
    return Math.max(report.baseline_points * (1 - offsetDays / totalSprintDays), 0);
  };

  const actualPath = points
    .map(
      (point, index) => `${index === 0 ? 'M' : 'L'} ${toX(index)} ${toY(point.remaining_points)}`
    )
    .join(' ');

  const idealPath = points
    .map((point, index) => {
      const idealValue = getIdealValue(point.date);
      return `${index === 0 ? 'M' : 'L'} ${toX(index)} ${toY(idealValue)}`;
    })
    .join(' ');

  const firstDate = points[0]?.date || '';
  const lastDate = points[points.length - 1]?.date || '';
  const currentRemaining = points[points.length - 1]?.remaining_points || 0;
  const idealRemaining = Number(getIdealValue(lastDate).toFixed(1));
  const burnedPoints = Math.max(report.baseline_points - currentRemaining, 0);

  return (
    <div>
      <div className="mb-3 grid grid-cols-1 gap-3 text-sm md:grid-cols-3">
        <div className="section-block rounded-[1.1rem] px-3 py-2">
          <div className="text-xs text-text-light">基线点数</div>
          <div className="font-semibold text-text">{report.baseline_points}</div>
        </div>
        <div className="section-block rounded-[1.1rem] px-3 py-2">
          <div className="text-xs text-text-light">当前剩余（实际）</div>
          <div className="font-semibold text-text">{currentRemaining}</div>
        </div>
        <div className="section-block rounded-[1.1rem] px-3 py-2">
          <div className="text-xs text-text-light">今日理想剩余</div>
          <div className="font-semibold text-text">{idealRemaining}</div>
        </div>
      </div>

      <svg viewBox={`0 0 ${width} ${height}`} className="h-64 w-full">
        {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
          const y = paddingY + plotHeight * ratio;
          return (
            <line
              key={ratio}
              x1={paddingX}
              y1={y}
              x2={paddingX + plotWidth}
              y2={y}
              stroke="#e2e8f0"
              strokeWidth="1"
            />
          );
        })}

        <path d={idealPath} fill="none" stroke="#94a3b8" strokeWidth="2" strokeDasharray="6 4" />
        <path d={actualPath} fill="none" stroke="#1e3a5f" strokeWidth="3" />

        {points.map((point, index) => (
          <circle
            key={`${point.date}-${index}`}
            cx={toX(index)}
            cy={toY(point.remaining_points)}
            r="3.5"
            fill="#1e3a5f"
          />
        ))}
      </svg>

      <div className="mt-3 flex flex-wrap items-center justify-between gap-3 text-xs text-text-light">
        <div className="flex items-center gap-4">
          <span className="inline-flex items-center gap-1">
            <span className="inline-block h-[2px] w-4 bg-slate-400" />
            理想线
          </span>
          <span className="inline-flex items-center gap-1">
            <span className="inline-block h-[2px] w-4 bg-primary" />
            实际线
          </span>
        </div>
        <div className="flex items-center gap-3">
          <span>{firstDate}</span>
          <span>→</span>
          <span>{lastDate}</span>
          <span>已燃尽: {burnedPoints} 点</span>
        </div>
      </div>

      <div className="mt-3 space-y-1 text-xs text-text-light">
        <div>理想线：基线点数在冲刺总天数内按线性下降计算，不会因为“今天”提前归零。</div>
        <div>实际线：到每天结束时，状态为“已完成”的故事点从基线中扣减后的剩余点数。</div>
        <div>统计范围：仅统计已规划到当前冲刺的故事。</div>
      </div>
    </div>
  );
}
