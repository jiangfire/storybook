import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import { bugService } from '../../services/bugService';
import { projectService } from '../../services/projectService';
import { storyService } from '../../services/storyService';
import { getErrorMessage } from '../../utils/error';
import { useToast } from '../../components/ui/Toast';
import { formatBugSeverity, formatBugStatus, formatDate } from '../../utils/formatters';
import {
  ArchiveIcon,
  BugIcon,
  CheckCircleIcon,
  ClipboardIcon,
  InboxIcon,
  StoryIcon,
  UsersIcon,
  WrenchIcon,
} from '../../components/ui/AppIcon';
import type { BugItem } from '../../types/api';

interface ProjectMemberOption {
  id: number;
  role_in_project: string;
  user_id: number;
  user?: {
    id: number;
    email: string;
  };
}

interface StoryOption {
  id: number;
  title: string;
}

const BUG_STATUS_OPTIONS: Array<{ value: BugItem['status']; label: string }> = [
  { value: 'open', label: '待处理' },
  { value: 'in_progress', label: '处理中' },
  { value: 'resolved', label: '已解决' },
  { value: 'closed', label: '已关闭' },
];

const BUG_SEVERITY_OPTIONS: Array<{ value: BugItem['severity']; label: string }> = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'critical', label: '严重' },
];

const bugStatusMeta: Record<
  BugItem['status'],
  { badgeClass: string; icon: typeof InboxIcon; cardClass: string }
> = {
  open: {
    badgeClass: 'bg-info-light text-info',
    icon: InboxIcon,
    cardClass: 'border-info/30 bg-white',
  },
  in_progress: {
    badgeClass: 'bg-warning-light text-warning',
    icon: WrenchIcon,
    cardClass: 'border-warning/30 bg-white',
  },
  resolved: {
    badgeClass: 'bg-success-light text-success',
    icon: CheckCircleIcon,
    cardClass: 'border-success/30 bg-white',
  },
  closed: {
    badgeClass: 'bg-secondary-100 text-text-light',
    icon: ArchiveIcon,
    cardClass: 'border-border bg-secondary-50',
  },
};

const bugSeverityMeta: Record<BugItem['severity'], { badgeClass: string }> = {
  low: { badgeClass: 'bg-success-light text-success' },
  medium: { badgeClass: 'bg-info-light text-info' },
  high: { badgeClass: 'bg-warning-light text-warning' },
  critical: { badgeClass: 'bg-danger-light text-danger' },
};

export default function ProjectBugsPage() {
  const { id } = useParams<{ id: string }>();
  const projectID = Number(id);
  const [searchParams, setSearchParams] = useSearchParams();
  const { showError, showSuccess } = useToast();

  const [bugs, setBugs] = useState<BugItem[]>([]);
  const [stories, setStories] = useState<StoryOption[]>([]);
  const [members, setMembers] = useState<ProjectMemberOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<BugItem | null>(null);

  const [filters, setFilters] = useState({
    status: '',
    severity: '',
    assignee: '',
  });

  const [form, setForm] = useState({
    title: '',
    description: '',
    severity: 'medium',
    story_id: '',
    assigned_to: '',
  });

  const assigneeOptions = useMemo(
    () =>
      members
        .filter((member) => !!member.user)
        .map((member) => ({
          id: member.user_id,
          email: member.user?.email || `用户 #${member.user_id}`,
          role_in_project: member.role_in_project,
        })),
    [members]
  );

  const storyTitleMap = useMemo(
    () => new Map(stories.map((story) => [story.id, story.title])),
    [stories]
  );

  const bugSummary = useMemo(
    () => ({
      total: bugs.length,
      open: bugs.filter((bug) => bug.status === 'open').length,
      progressing: bugs.filter((bug) => bug.status === 'in_progress').length,
      critical: bugs.filter((bug) => bug.severity === 'critical').length,
      unassigned: bugs.filter((bug) => !bug.assigned_to).length,
    }),
    [bugs]
  );

  const activeFilterCount = [filters.status, filters.severity, filters.assignee].filter(Boolean)
    .length;

  const loadBaseData = useCallback(async () => {
    if (Number.isNaN(projectID) || projectID <= 0) {
      return;
    }
    try {
      const [memberData, storyData] = await Promise.all([
        projectService.getProjectMembers(projectID),
        storyService.getStories(projectID),
      ]);
      setMembers(memberData.members || []);
      setStories((storyData.stories || []).map((item) => ({ id: item.id, title: item.title })));
    } catch (err: unknown) {
      showError(getErrorMessage(err, '基础数据加载失败'));
    }
  }, [projectID, showError]);

  const loadBugs = useCallback(async () => {
    if (Number.isNaN(projectID) || projectID <= 0) {
      return;
    }
    try {
      setLoading(true);
      setError('');
      const data = await bugService.getProjectBugs(projectID, {
        status: filters.status || undefined,
        severity: filters.severity || undefined,
        assignee: filters.assignee ? Number(filters.assignee) : undefined,
      });
      setBugs(data.bugs || []);
    } catch (err: unknown) {
      setError(getErrorMessage(err, '缺陷列表加载失败'));
      setBugs([]);
    } finally {
      setLoading(false);
    }
  }, [filters.assignee, filters.severity, filters.status, projectID]);

  useEffect(() => {
    void loadBaseData();
  }, [loadBaseData]);

  useEffect(() => {
    void loadBugs();
  }, [loadBugs]);

  const openBugDetail = useCallback(
    async (bugID: number) => {
      try {
        setDetailOpen(true);
        setLoadingDetail(true);
        const data = await bugService.getBug(bugID);
        setDetail(data);
      } catch (err: unknown) {
        showError(getErrorMessage(err, '缺陷详情加载失败'));
        setDetailOpen(false);
      } finally {
        setLoadingDetail(false);
      }
    },
    [showError]
  );

  const bugQueryParam = searchParams.get('bug');

  useEffect(() => {
    const bugID = bugQueryParam;
    if (!bugID) {
      return;
    }
    const parsed = Number(bugID);
    if (!Number.isNaN(parsed) && parsed > 0) {
      void openBugDetail(parsed);
    }
  }, [bugQueryParam, openBugDetail]);

  const closeBugDetail = useCallback(() => {
    setDetailOpen(false);
    setDetail(null);

    if (!bugQueryParam) {
      return;
    }

    const nextParams = new URLSearchParams(searchParams);
    nextParams.delete('bug');
    setSearchParams(nextParams, { replace: true });
  }, [bugQueryParam, searchParams, setSearchParams]);

  const handleCreate = async () => {
    const title = form.title.trim();
    if (!title) {
      showError('请输入缺陷标题');
      return;
    }
    try {
      setCreating(true);
      await bugService.createBug(projectID, {
        title,
        description: form.description.trim() || undefined,
        severity: form.severity as 'low' | 'medium' | 'high' | 'critical',
        story_id: form.story_id ? Number(form.story_id) : undefined,
        assigned_to: form.assigned_to ? Number(form.assigned_to) : undefined,
      });
      setForm({
        title: '',
        description: '',
        severity: 'medium',
        story_id: '',
        assigned_to: '',
      });
      showSuccess('缺陷创建成功');
      await loadBugs();
    } catch (err: unknown) {
      showError(getErrorMessage(err, '缺陷创建失败'));
    } finally {
      setCreating(false);
    }
  };

  const handleStatusChange = async (
    bugID: number,
    status: 'open' | 'in_progress' | 'resolved' | 'closed'
  ) => {
    try {
      await bugService.updateBugStatus(bugID, { status });
      setBugs((prev) => prev.map((item) => (item.id === bugID ? { ...item, status } : item)));
      if (detail?.id === bugID) {
        setDetail((prev) => (prev ? { ...prev, status } : prev));
      }
    } catch (err: unknown) {
      showError(getErrorMessage(err, '缺陷状态更新失败'));
    }
  };

  const handleAssign = async (bugID: number, assignedTo: string) => {
    try {
      const payload = assignedTo ? Number(assignedTo) : undefined;
      const updated = await bugService.assignBug(bugID, {
        assigned_to: payload,
      });
      setBugs((prev) =>
        prev.map((item) =>
          item.id === bugID
            ? {
                ...item,
                assigned_to: updated.assigned_to,
              }
            : item
        )
      );
      if (detail?.id === bugID) {
        setDetail((prev) => (prev ? { ...prev, assigned_to: updated.assigned_to } : prev));
      }
      showSuccess('缺陷指派已更新');
    } catch (err: unknown) {
      showError(getErrorMessage(err, '缺陷指派失败'));
    }
  };

  const renderAssignedTo = (bug: BugItem) => {
    if (!bug.assigned_to) {
      return '未指派';
    }
    if (typeof bug.assigned_to === 'number') {
      const matched = assigneeOptions.find((option) => option.id === bug.assigned_to);
      return matched?.email || `用户 #${bug.assigned_to}`;
    }
    return bug.assigned_to.email;
  };

  const getStatusMeta = (status: BugItem['status']) => bugStatusMeta[status];
  const getSeverityMeta = (severity: BugItem['severity']) => bugSeverityMeta[severity];

  const handleResetFilters = () => {
    setFilters({
      status: '',
      severity: '',
      assignee: '',
    });
  };

  if (Number.isNaN(projectID) || projectID <= 0) {
    return <div className="p-8 text-danger">项目ID无效</div>;
  }

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <section className="surface-card overflow-hidden rounded-[2rem]">
        <div className="grid gap-5 px-5 py-6 sm:px-6 lg:grid-cols-[minmax(0,1fr)_minmax(280px,420px)] lg:px-8 lg:py-8">
          <div className="space-y-4">
            <span className="inline-flex items-center gap-2 rounded-full bg-danger-light px-3 py-1 text-xs font-medium text-danger">
              <BugIcon size={14} />
              缺陷管理
            </span>
            <div className="space-y-2">
              <div className="flex flex-wrap items-center gap-2 text-sm text-text-light">
                <Link to={`/projects/${projectID}`} className="transition-colors hover:text-primary">
                  项目详情
                </Link>
                <span>›</span>
                <span className="text-text">缺陷管理</span>
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-text sm:text-4xl">
                  缺陷管理
                </h1>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-text-light sm:text-base">
                  先筛出高风险问题，再明确责任并推进关闭。
                </p>
              </div>
            </div>
          </div>

          <div className="rounded-[1.6rem] border border-primary-700 bg-primary-800 p-5 text-white shadow-md">
            <div className="text-xs font-medium text-white/70">筛选重点</div>
            <div className="mt-3 text-2xl font-semibold">{activeFilterCount} 个筛选条件生效</div>
            <p className="mt-2 text-sm leading-6 text-white/80">
              {activeFilterCount > 0
                ? '当前列表已经收拢到重点问题，可继续刷新或清空筛选。'
                : '建议优先关注待处理、高严重级别和未指派问题。'}
            </p>
            <div className="mt-4 flex flex-wrap gap-2 text-xs">
              <span className="rounded-full bg-white/15 px-3 py-1">待处理 {bugSummary.open}</span>
              <span className="rounded-full bg-white/15 px-3 py-1">严重 {bugSummary.critical}</span>
              <span className="rounded-full bg-white/15 px-3 py-1">
                未指派 {bugSummary.unassigned}
              </span>
            </div>
          </div>
        </div>
      </section>

      <section className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div className="surface-card rounded-[1.5rem] p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-text-light">
            <BugIcon size={14} />
            全部
          </div>
          <div className="mt-3 text-3xl font-semibold text-text">{bugSummary.total}</div>
        </div>
        <div className="surface-card rounded-[1.5rem] p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-text-light">
            <InboxIcon size={14} />
            待处理
          </div>
          <div className="mt-3 text-3xl font-semibold text-info">{bugSummary.open}</div>
        </div>
        <div className="surface-card rounded-[1.5rem] p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-text-light">
            <WrenchIcon size={14} />
            处理中
          </div>
          <div className="mt-3 text-3xl font-semibold text-warning">{bugSummary.progressing}</div>
        </div>
        <div className="surface-card rounded-[1.5rem] p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-text-light">
            <UsersIcon size={14} />
            未指派
          </div>
          <div className="mt-3 text-3xl font-semibold text-text">{bugSummary.unassigned}</div>
        </div>
      </section>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
        <div className="surface-card rounded-[1.8rem] p-4 sm:p-5">
          <div className="mb-4 flex items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-text">新建缺陷</h2>
              <p className="mt-1 text-sm text-text-light">先把问题记清楚，再决定归属。</p>
            </div>
            <span className="inline-flex items-center gap-1 rounded-full bg-danger-light px-2.5 py-1 text-xs font-medium text-danger">
              <BugIcon size={12} />
              严重 {bugSummary.critical}
            </span>
          </div>
          <div className="space-y-3">
            <div>
              <label className="mb-2 block text-sm font-medium text-text">标题</label>
              <input
                value={form.title}
                onChange={(e) => setForm((prev) => ({ ...prev, title: e.target.value }))}
                placeholder="一句话说清问题"
                className="field-control"
              />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-text">描述</label>
              <textarea
                value={form.description}
                onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
                placeholder="补充复现方式、影响范围或截图说明"
                rows={4}
                className="field-control"
              />
            </div>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
              <div>
                <label className="mb-2 block text-sm font-medium text-text">严重级别</label>
                <select
                  value={form.severity}
                  onChange={(e) => setForm((prev) => ({ ...prev, severity: e.target.value }))}
                  className="field-control"
                >
                  {BUG_SEVERITY_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-2 block text-sm font-medium text-text">关联故事</label>
                <select
                  value={form.story_id}
                  onChange={(e) => setForm((prev) => ({ ...prev, story_id: e.target.value }))}
                  className="field-control"
                >
                  <option value="">暂不关联</option>
                  {stories.map((story) => (
                    <option key={story.id} value={story.id}>
                      #{story.id} {story.title}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-2 block text-sm font-medium text-text">初始负责人</label>
                <select
                  value={form.assigned_to}
                  onChange={(e) => setForm((prev) => ({ ...prev, assigned_to: e.target.value }))}
                  className="field-control"
                >
                  <option value="">暂不指派</option>
                  {assigneeOptions.map((member) => (
                    <option key={member.id} value={member.id}>
                      {member.email}（{member.role_in_project}）
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <Button size="sm" onClick={handleCreate} isLoading={creating}>
              创建缺陷
            </Button>
          </div>
        </div>

        <div className="surface-card rounded-[1.8rem] p-4 sm:p-5">
          <div className="mb-4">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-lg font-semibold text-text">筛选</h2>
              {activeFilterCount > 0 && (
                <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary">
                  {activeFilterCount} 个筛选
                </span>
              )}
            </div>
            <p className="mt-1 text-sm text-text-light">优先把待处理和高严重级别问题收拢出来。</p>
          </div>
          <div className="space-y-3">
            <div>
              <label className="mb-2 block text-sm font-medium text-text">状态</label>
              <select
                value={filters.status}
                onChange={(e) => setFilters((prev) => ({ ...prev, status: e.target.value }))}
                className="field-control"
              >
                <option value="">全部状态</option>
                {BUG_STATUS_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-text">严重级别</label>
              <select
                value={filters.severity}
                onChange={(e) => setFilters((prev) => ({ ...prev, severity: e.target.value }))}
                className="field-control"
              >
                <option value="">全部级别</option>
                {BUG_SEVERITY_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-text">负责人</label>
              <select
                value={filters.assignee}
                onChange={(e) => setFilters((prev) => ({ ...prev, assignee: e.target.value }))}
                className="field-control"
              >
                <option value="">全部负责人</option>
                {assigneeOptions.map((member) => (
                  <option key={member.id} value={member.id}>
                    {member.email}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap justify-end gap-2">
            <Button size="sm" variant="ghost" onClick={handleResetFilters} disabled={loading}>
              清空
            </Button>
            <Button size="sm" variant="secondary" onClick={() => void loadBugs()} disabled={loading}>
              刷新列表
            </Button>
          </div>
        </div>
      </div>

      {error && <div className="state-panel state-panel-error">{error}</div>}
      {loading ? (
        <div className="state-panel state-panel-loading">
          缺陷列表加载中...
        </div>
      ) : bugs.length === 0 ? (
        <div className="state-panel state-panel-empty">
          当前筛选下暂无缺陷
        </div>
      ) : (
        <section className="space-y-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-text">缺陷列表</h2>
              <p className="mt-1 text-sm text-text-light">按状态和责任人推进处理，必要时直接进入详情。</p>
            </div>
            {activeFilterCount > 0 && (
              <div className="flex flex-wrap gap-2 text-xs text-text-light">
                {filters.status && (
                  <span className="rounded-full bg-secondary-50 px-2.5 py-1">
                    状态：{formatBugStatus(filters.status as BugItem['status'])}
                  </span>
                )}
                {filters.severity && (
                  <span className="rounded-full bg-secondary-50 px-2.5 py-1">
                    级别：{formatBugSeverity(filters.severity as BugItem['severity'])}
                  </span>
                )}
                {filters.assignee && (
                  <span className="rounded-full bg-secondary-50 px-2.5 py-1">负责人已筛选</span>
                )}
              </div>
            )}
          </div>
          <div className="space-y-3">
          {bugs.map((bug) => {
            const statusMeta = getStatusMeta(bug.status);
            const severityMeta = getSeverityMeta(bug.severity);
            const StatusIcon = statusMeta.icon;

            return (
              <div
                key={bug.id}
                className={`card-hover rounded-[1.6rem] border p-4 shadow-sm sm:p-5 ${statusMeta.cardClass}`}
              >
                <div className="flex flex-col gap-4">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-semibold text-text">#{bug.id}</span>
                        <span
                          className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium ${statusMeta.badgeClass}`}
                        >
                          <StatusIcon size={12} />
                          {formatBugStatus(bug.status)}
                        </span>
                        <span
                          className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${severityMeta.badgeClass}`}
                        >
                          {formatBugSeverity(bug.severity)}
                        </span>
                      </div>
                      <div className="mt-2 text-base font-semibold text-text sm:text-lg">
                        {bug.title}
                      </div>
                      <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-text-light">
                        <span className="inline-flex items-center gap-1">
                          <UsersIcon size={12} />
                          {renderAssignedTo(bug)}
                        </span>
                        <span className="inline-flex items-center gap-1">
                          <ClipboardIcon size={12} />
                          更新于 {formatDate(bug.updated_at)}
                        </span>
                        {bug.story_id && (
                          <span className="inline-flex items-center gap-1">
                            <StoryIcon size={12} />
                            故事 #{bug.story_id}
                            {storyTitleMap.get(bug.story_id)
                              ? ` · ${storyTitleMap.get(bug.story_id)}`
                              : ''}
                          </span>
                        )}
                      </div>
                      {bug.description && (
                        <p className="mt-3 line-clamp-2 text-sm leading-6 text-text-light">{bug.description}</p>
                      )}
                    </div>
                    <div className="flex w-full flex-col gap-2 sm:w-auto sm:min-w-[220px]">
                      <select
                        value={bug.status}
                        onChange={(e) =>
                          void handleStatusChange(
                            bug.id,
                            e.target.value as 'open' | 'in_progress' | 'resolved' | 'closed'
                          )
                        }
                        className="field-control"
                      >
                        {BUG_STATUS_OPTIONS.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </select>
                      <select
                        value={
                          bug.assigned_to
                            ? typeof bug.assigned_to === 'number'
                              ? String(bug.assigned_to)
                              : String(bug.assigned_to.id)
                            : ''
                        }
                        onChange={(e) => void handleAssign(bug.id, e.target.value)}
                        className="field-control"
                      >
                        <option value="">未指派</option>
                        {assigneeOptions.map((member) => (
                          <option key={member.id} value={member.id}>
                            {member.email}
                          </option>
                        ))}
                      </select>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => void openBugDetail(bug.id)}
                      >
                        查看详情
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
          </div>
        </section>
      )}

      <Modal isOpen={detailOpen} onClose={closeBugDetail} title="缺陷详情" size="md">
        {loadingDetail || !detail ? (
          <div className="state-panel state-panel-loading">加载详情中...</div>
        ) : (
          <div className="space-y-2 text-sm">
            <div>
              <span className="text-text-light">ID：</span>#{detail.id}
            </div>
            <div>
              <span className="text-text-light">标题：</span>
              {detail.title}
            </div>
            <div>
              <span className="text-text-light">状态：</span>
              {formatBugStatus(detail.status)}
            </div>
            <div>
              <span className="text-text-light">严重级别：</span>
              {formatBugSeverity(detail.severity)}
            </div>
            {detail.story_id && (
              <div>
                <span className="text-text-light">关联故事：</span>#{detail.story_id}
              </div>
            )}
            {detail.description && (
              <div>
                <span className="text-text-light">描述：</span>
                <div className="mt-1 whitespace-pre-wrap text-text">{detail.description}</div>
              </div>
            )}
            <div>
              <span className="text-text-light">创建时间：</span>
              {new Date(detail.created_at).toLocaleString()}
            </div>
            <div>
              <span className="text-text-light">更新时间：</span>
              {new Date(detail.updated_at).toLocaleString()}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
