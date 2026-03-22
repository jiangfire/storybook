import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import { bugService } from '../../services/bugService';
import { projectService } from '../../services/projectService';
import { storyService } from '../../services/storyService';
import { getErrorMessage } from '../../utils/error';
import { useToast } from '../../components/ui/Toast';
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

  if (Number.isNaN(projectID) || projectID <= 0) {
    return <div className="p-8 text-danger">项目ID无效</div>;
  }

  return (
    <div className="p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text mb-2">缺陷管理</h1>
        <p className="text-text-light">项目 #{projectID} 的缺陷跟踪与流转</p>
      </div>

      <div className="bg-white rounded-xl border border-border p-4 space-y-3">
        <h2 className="font-medium text-text">新建缺陷</h2>
        <input
          value={form.title}
          onChange={(e) => setForm((prev) => ({ ...prev, title: e.target.value }))}
          placeholder="缺陷标题"
          className="w-full px-3 py-2 border border-border rounded-lg"
        />
        <textarea
          value={form.description}
          onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
          placeholder="缺陷描述（可选）"
          rows={3}
          className="w-full px-3 py-2 border border-border rounded-lg resize-none"
        />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
          <select
            value={form.severity}
            onChange={(e) => setForm((prev) => ({ ...prev, severity: e.target.value }))}
            className="px-3 py-2 border border-border rounded-lg"
          >
            <option value="low">低</option>
            <option value="medium">中</option>
            <option value="high">高</option>
            <option value="critical">严重</option>
          </select>
          <select
            value={form.story_id}
            onChange={(e) => setForm((prev) => ({ ...prev, story_id: e.target.value }))}
            className="px-3 py-2 border border-border rounded-lg"
          >
            <option value="">关联故事（可选）</option>
            {stories.map((story) => (
              <option key={story.id} value={story.id}>
                #{story.id} {story.title}
              </option>
            ))}
          </select>
          <select
            value={form.assigned_to}
            onChange={(e) => setForm((prev) => ({ ...prev, assigned_to: e.target.value }))}
            className="px-3 py-2 border border-border rounded-lg"
          >
            <option value="">初始指派（可选）</option>
            {assigneeOptions.map((member) => (
              <option key={member.id} value={member.id}>
                {member.email}（{member.role_in_project}）
              </option>
            ))}
          </select>
        </div>
        <div className="flex justify-end">
          <Button size="sm" onClick={handleCreate} isLoading={creating}>
            创建缺陷
          </Button>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-border p-4 space-y-3">
        <h2 className="font-medium text-text">筛选</h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-2">
          <select
            value={filters.status}
            onChange={(e) => setFilters((prev) => ({ ...prev, status: e.target.value }))}
            className="px-3 py-2 border border-border rounded-lg"
          >
            <option value="">全部状态</option>
            <option value="open">open</option>
            <option value="in_progress">in_progress</option>
            <option value="resolved">resolved</option>
            <option value="closed">closed</option>
          </select>
          <select
            value={filters.severity}
            onChange={(e) => setFilters((prev) => ({ ...prev, severity: e.target.value }))}
            className="px-3 py-2 border border-border rounded-lg"
          >
            <option value="">全部严重级别</option>
            <option value="low">low</option>
            <option value="medium">medium</option>
            <option value="high">high</option>
            <option value="critical">critical</option>
          </select>
          <select
            value={filters.assignee}
            onChange={(e) => setFilters((prev) => ({ ...prev, assignee: e.target.value }))}
            className="px-3 py-2 border border-border rounded-lg"
          >
            <option value="">全部负责人</option>
            {assigneeOptions.map((member) => (
              <option key={member.id} value={member.id}>
                {member.email}
              </option>
            ))}
          </select>
          <Button size="sm" variant="secondary" onClick={() => void loadBugs()} disabled={loading}>
            刷新列表
          </Button>
        </div>
      </div>

      {error && <div className="text-sm text-danger">{error}</div>}
      {loading ? (
        <div className="text-sm text-text-light">缺陷列表加载中...</div>
      ) : bugs.length === 0 ? (
        <div className="text-sm text-text-light">暂无缺陷</div>
      ) : (
        <div className="space-y-3">
          {bugs.map((bug) => (
            <div key={bug.id} className="bg-white rounded-xl border border-border p-4">
              <div className="flex flex-col lg:flex-row lg:items-center gap-3">
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-text truncate">
                    #{bug.id} {bug.title}
                  </div>
                  <div className="text-xs text-text-light">
                    {bug.severity} · 当前负责人：{renderAssignedTo(bug)}
                  </div>
                </div>
                <select
                  value={bug.status}
                  onChange={(e) =>
                    void handleStatusChange(
                      bug.id,
                      e.target.value as 'open' | 'in_progress' | 'resolved' | 'closed'
                    )
                  }
                  className="px-2 py-1 border border-border rounded text-sm"
                >
                  <option value="open">open</option>
                  <option value="in_progress">in_progress</option>
                  <option value="resolved">resolved</option>
                  <option value="closed">closed</option>
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
                  className="px-2 py-1 border border-border rounded text-sm"
                >
                  <option value="">未指派</option>
                  {assigneeOptions.map((member) => (
                    <option key={member.id} value={member.id}>
                      {member.email}
                    </option>
                  ))}
                </select>
                <Button size="sm" variant="secondary" onClick={() => void openBugDetail(bug.id)}>
                  详情
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Modal isOpen={detailOpen} onClose={closeBugDetail} title="缺陷详情" size="md">
        {loadingDetail || !detail ? (
          <div className="text-sm text-text-light">加载详情中...</div>
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
              {detail.status}
            </div>
            <div>
              <span className="text-text-light">严重级别：</span>
              {detail.severity}
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
