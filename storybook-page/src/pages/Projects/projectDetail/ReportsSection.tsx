import Button from '../../../components/ui/Button';
import {
  ArchiveIcon,
  CheckCircleIcon,
  ClipboardIcon,
  InboxIcon,
  WrenchIcon,
} from '../../../components/ui/AppIcon';
import { formatSprintStatus, formatStoryStatus } from '../../../utils/formatters';
import type {
  CumulativeFlowReport,
  QualityReportData,
  TimeMetricReport,
  ThroughputReport,
  VelocityReportData,
} from '../../../types/api';

interface ReportsSectionProps {
  isReportLoading: boolean;
  reportError: string;
  velocity: VelocityReportData | null;
  quality: QualityReportData | null;
  cumulativeFlow: CumulativeFlowReport | null;
  cycleTime: TimeMetricReport | null;
  leadTime: TimeMetricReport | null;
  throughput: ThroughputReport | null;
  onRefresh: () => void;
}

function BugStatusDistribution({ data, total }: { data: Record<string, number>; total: number }) {
  const statusConfig: Record<
    string,
    {
      label: string;
      icon: typeof InboxIcon;
      color: string;
      bgColor: string;
    }
  > = {
    open: { label: '待处理', icon: InboxIcon, color: 'text-info', bgColor: 'bg-info-light' },
    in_progress: {
      label: '处理中',
      icon: WrenchIcon,
      color: 'text-warning',
      bgColor: 'bg-warning-light',
    },
    resolved: {
      label: '已解决',
      icon: CheckCircleIcon,
      color: 'text-success',
      bgColor: 'bg-success-light',
    },
    closed: {
      label: '已关闭',
      icon: ArchiveIcon,
      color: 'text-text-light',
      bgColor: 'bg-secondary-100',
    },
  };

  const entries = Object.entries(data).sort((a, b) => b[1] - a[1]);
  const maxValue = Math.max(...entries.map(([, value]) => value), 1);

  return (
    <div>
      <div className="mb-3 flex items-center justify-between text-xs text-text-light">
        <span>缺陷状态分布</span>
        <span>{total} 个</span>
      </div>
      <div className="space-y-2">
        {entries.map(([key, value]) => {
          const config = statusConfig[key] || {
            label: key,
            icon: ClipboardIcon,
            color: 'text-text',
            bgColor: 'bg-secondary-100',
          };
          const percentage = total > 0 ? (value / total) * 100 : 0;
          const barWidth = maxValue > 0 ? (value / maxValue) * 100 : 0;
          const StatusIcon = config.icon;

          return (
            <div key={key} className="group">
              <div className="mb-1 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span
                    className={`flex h-6 w-6 items-center justify-center rounded-md text-sm ${config.bgColor}`}
                  >
                    <StatusIcon size={14} />
                  </span>
                  <span className="text-sm text-text">{config.label}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`text-sm font-semibold ${config.color}`}>{value}</span>
                  {total > 0 && (
                    <span className="w-10 text-right text-xs text-text-light">
                      {percentage.toFixed(0)}%
                    </span>
                  )}
                </div>
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-secondary-100">
                <div
                  className={`h-full rounded-full transition-all duration-500 ${config.bgColor.replace('-light', '')}`}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function BugSeverityDistribution({ data, total }: { data: Record<string, number>; total: number }) {
  const severityConfig: Record<string, { label: string; color: string; dotColor: string }> = {
    critical: { label: '严重', color: 'text-danger', dotColor: 'bg-danger' },
    high: { label: '高', color: 'text-warning', dotColor: 'bg-warning' },
    medium: { label: '中', color: 'text-info', dotColor: 'bg-info' },
    low: { label: '低', color: 'text-success', dotColor: 'bg-success' },
  };

  const entries = Object.entries(data).sort((a, b) => {
    const order = ['critical', 'high', 'medium', 'low'];
    return order.indexOf(a[0]) - order.indexOf(b[0]);
  });

  return (
    <div>
      <div className="mb-3 text-xs text-text-light">缺陷严重级别分布</div>
      <div className="grid grid-cols-2 gap-2">
        {entries.map(([key, value]) => {
          const config = severityConfig[key] || {
            label: key,
            color: 'text-text',
            dotColor: 'bg-secondary-400',
          };
          const percentage = total > 0 ? (value / total) * 100 : 0;

          return (
            <div key={key} className="rounded-lg border border-border bg-white p-3">
              <div className="flex items-start justify-between">
                <span className="inline-flex items-center gap-2">
                  <span className={`h-2.5 w-2.5 rounded-full ${config.dotColor}`} />
                  <span className={`text-sm font-medium ${config.color}`}>{config.label}</span>
                </span>
                <span className={`text-2xl font-bold ${config.color}`}>{value}</span>
              </div>
              {total > 0 && <div className="mt-2 text-xs text-text-light">{percentage.toFixed(1)}%</div>}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function ReportsSection({
  isReportLoading,
  reportError,
  velocity,
  quality,
  cumulativeFlow,
  cycleTime,
  leadTime,
  throughput,
  onRefresh,
}: ReportsSectionProps) {
  const latestFlowPoint = cumulativeFlow?.points[cumulativeFlow.points.length - 1] ?? null;
  const flowEntries = latestFlowPoint
    ? Object.entries(latestFlowPoint.statuses).filter(([, count]) => count > 0)
    : [];
  const flowMax = Math.max(...flowEntries.map(([, count]) => count), 1);
  const throughputMax = Math.max(...(throughput?.points.map((p) => p.completed_count) ?? []), 1);

  return (
    <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
      <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-text">速度报表</h2>
            <p className="mt-1 text-sm text-text-light">用已完成点数和计划点数判断团队交付节奏。</p>
          </div>
          <Button size="sm" variant="secondary" onClick={onRefresh}>
            刷新
          </Button>
        </div>
        {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
        {!isReportLoading && reportError && <div className="state-panel state-panel-error">{reportError}</div>}
        {!isReportLoading && !reportError && (!velocity || velocity.velocity.length === 0) && (
          <div className="state-panel state-panel-empty">暂无冲刺速度数据</div>
        )}
        {!isReportLoading && !reportError && velocity && velocity.velocity.length > 0 && (
          <div className="space-y-2">
            {velocity.velocity.map((item) => (
              <div key={item.sprint_id} className="section-block rounded-[1.2rem] p-3">
                <div className="font-medium text-text">{item.name}</div>
                <div className="mt-1 text-xs text-text-light">
                  状态：{formatSprintStatus(item.status)} · 完成点数 {item.completed_points}/
                  {item.planned_points} · 速度 {item.velocity.toFixed(1)}%
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-text">质量报表</h2>
          <p className="mt-1 text-sm text-text-light">结合缺陷和 AC 完成率看当前质量风险。</p>
        </div>
        {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
        {!isReportLoading && reportError && <div className="state-panel state-panel-error">{reportError}</div>}
        {!isReportLoading && !reportError && quality && (
          <div className="space-y-3 text-sm">
            <div className="grid grid-cols-2 gap-3">
              <div className="section-block rounded-[1.2rem] p-3">
                <div className="text-xs text-text-light">缺陷总数</div>
                <div className="mt-1 text-xl font-semibold text-text">{quality.bugs.total}</div>
              </div>
              <div className="section-block rounded-[1.2rem] p-3">
                <div className="text-xs text-text-light">AC 完成率</div>
                <div className="mt-1 text-xl font-semibold text-text">
                  {quality.acceptance_criteria.completion_percentage.toFixed(1)}%
                </div>
              </div>
            </div>

            <BugStatusDistribution data={quality.bugs.status_breakdown} total={quality.bugs.total} />
            <BugSeverityDistribution
              data={quality.bugs.severity_breakdown}
              total={quality.bugs.total}
            />
          </div>
        )}
      </div>

      {/* 累计流 */}
      <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-text">累计流</h2>
          <p className="mt-1 text-sm text-text-light">
            {cumulativeFlow ? `${cumulativeFlow.from} 至 ${cumulativeFlow.to} 每日各状态存量。` : '每日各状态故事存量。'}
          </p>
        </div>
        {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
        {!isReportLoading && reportError && <div className="state-panel state-panel-error">{reportError}</div>}
        {!isReportLoading && !reportError && (!cumulativeFlow || flowEntries.length === 0) && (
          <div className="state-panel state-panel-empty">暂无累计流数据</div>
        )}
        {!isReportLoading && !reportError && cumulativeFlow && flowEntries.length > 0 && (
          <div className="space-y-2">
            <div className="text-xs text-text-light">截至 {latestFlowPoint?.date} 的状态分布</div>
            {flowEntries.map(([status, count]) => (
              <div key={status} className="flex items-center gap-3">
                <span className="w-16 shrink-0 text-xs text-text-light">
                  {formatStoryStatus(status)}
                </span>
                <div className="h-2 flex-1 rounded-full bg-secondary-100">
                  <div
                    className="h-2 rounded-full bg-primary"
                    style={{ width: `${(count / flowMax) * 100}%` }}
                  />
                </div>
                <span className="w-8 shrink-0 text-right text-xs text-text">{count}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 吞吐量 */}
      <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-text">吞吐量</h2>
          <p className="mt-1 text-sm text-text-light">
            {throughput
              ? `按${throughput.interval === 'day' ? '日' : '周'}统计完成故事数，窗口内共完成 ${throughput.total_completed} 个。`
              : '按周期统计完成故事数。'}
          </p>
        </div>
        {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
        {!isReportLoading && reportError && <div className="state-panel state-panel-error">{reportError}</div>}
        {!isReportLoading && !reportError && (!throughput || throughput.total_completed === 0) && (
          <div className="state-panel state-panel-empty">暂无吞吐量数据</div>
        )}
        {!isReportLoading && !reportError && throughput && throughput.total_completed > 0 && (
          <div className="space-y-2">
            {throughput.points.map((point) => (
              <div key={point.period_start} className="flex items-center gap-3">
                <span className="w-24 shrink-0 text-xs text-text-light">{point.period_start}</span>
                <div className="h-2 flex-1 rounded-full bg-secondary-100">
                  <div
                    className="h-2 rounded-full bg-success"
                    style={{ width: `${(point.completed_count / throughputMax) * 100}%` }}
                  />
                </div>
                <span className="w-8 shrink-0 text-right text-xs text-text">
                  {point.completed_count}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 周期时间 */}
      <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-text">周期时间</h2>
          <p className="mt-1 text-sm text-text-light">从开始开发到完成的平均耗时。</p>
        </div>
        {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
        {!isReportLoading && reportError && <div className="state-panel state-panel-error">{reportError}</div>}
        {!isReportLoading && !reportError && (!cycleTime || cycleTime.sample_size === 0) && (
          <div className="state-panel state-panel-empty">暂无周期时间数据</div>
        )}
        {!isReportLoading && !reportError && cycleTime && cycleTime.sample_size > 0 && (
          <div className="space-y-2 text-sm">
            <div className="grid grid-cols-2 gap-3">
              <div className="section-block rounded-[1.2rem] p-3">
                <div className="text-xs text-text-light">平均周期</div>
                <div className="mt-1 text-xl font-semibold text-text">
                  {cycleTime.average_days.toFixed(1)} 天
                </div>
              </div>
              <div className="section-block rounded-[1.2rem] p-3">
                <div className="text-xs text-text-light">样本数</div>
                <div className="mt-1 text-xl font-semibold text-text">{cycleTime.sample_size}</div>
              </div>
            </div>
            <div className="text-xs text-text-light">耗时最长的故事：</div>
            {cycleTime.per_story.slice(0, 3).map((item) => (
              <div key={item.story_id} className="section-block rounded-[1.2rem] px-3 py-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate">#{item.story_id} {item.title}</span>
                  <span className="shrink-0 text-xs text-text-light">
                    {item.days.toFixed(1)} 天
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 前置时间 */}
      <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-text">前置时间</h2>
          <p className="mt-1 text-sm text-text-light">从故事创建到完成的平均耗时。</p>
        </div>
        {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
        {!isReportLoading && reportError && <div className="state-panel state-panel-error">{reportError}</div>}
        {!isReportLoading && !reportError && (!leadTime || leadTime.sample_size === 0) && (
          <div className="state-panel state-panel-empty">暂无前置时间数据</div>
        )}
        {!isReportLoading && !reportError && leadTime && leadTime.sample_size > 0 && (
          <div className="space-y-2 text-sm">
            <div className="grid grid-cols-2 gap-3">
              <div className="section-block rounded-[1.2rem] p-3">
                <div className="text-xs text-text-light">平均前置</div>
                <div className="mt-1 text-xl font-semibold text-text">
                  {leadTime.average_days.toFixed(1)} 天
                </div>
              </div>
              <div className="section-block rounded-[1.2rem] p-3">
                <div className="text-xs text-text-light">样本数</div>
                <div className="mt-1 text-xl font-semibold text-text">{leadTime.sample_size}</div>
              </div>
            </div>
            <div className="text-xs text-text-light">耗时最长的故事：</div>
            {leadTime.per_story.slice(0, 3).map((item) => (
              <div key={item.story_id} className="section-block rounded-[1.2rem] px-3 py-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate">#{item.story_id} {item.title}</span>
                  <span className="shrink-0 text-xs text-text-light">
                    {item.days.toFixed(1)} 天
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
