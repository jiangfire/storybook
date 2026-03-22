import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useProjectStore } from '../../stores/projectStore';
import { useAuthStore } from '../../stores/authStore';
import { projectService } from '../../services/projectService';
import { techLeadService } from '../../services/techLeadService';
import { userManagementService } from '../../services/userManagementService';
import Button from '../../components/ui/Button';
import Modal from '../../components/ui/Modal';
import { useToast } from '../../components/ui/Toast';
import { formatStoryStatus, formatSprintStatus } from '../../utils/formatters';
import { getErrorMessage } from '../../utils/error';
import type {
  BurndownReport,
  CreateSprintRequest,
  QualityReportData,
  SprintSummary,
  VelocityReportData,
} from '../../types/api';
import type { User } from '../../types/models';
import {
  ArchiveIcon,
  BoardIcon,
  CheckCircleIcon,
  ClipboardIcon,
  CodeIcon,
  CompassIcon,
  CrownIcon,
  InboxIcon,
  SearchIcon,
  SprintIcon,
  StoryIcon,
  UsersIcon,
  WrenchIcon,
} from '../../components/ui/AppIcon';

interface MemberItem {
  id: number;
  user_id: number;
  role_in_project: string;
  joined_at: string;
  is_owner: boolean;
  user?: {
    id: number;
    email: string;
    avatar_url?: string;
  };
}

type ConfirmActionState =
  | {
      kind: 'remove_member';
      title: string;
      message: string;
      userID: number;
    }
  | {
      kind: 'remove_tech_lead';
      title: string;
      message: string;
      userID: number;
    };

const memberRoleMeta: Record<
  string,
  { icon: typeof ClipboardIcon; label: string; colorClass: string; bgClass: string }
> = {
  product: {
    icon: ClipboardIcon,
    label: '产品经理',
    colorClass: 'text-blue-700',
    bgClass: 'bg-blue-50',
  },
  developer: {
    icon: CodeIcon,
    label: '开发',
    colorClass: 'text-emerald-700',
    bgClass: 'bg-emerald-50',
  },
  tester: {
    icon: SearchIcon,
    label: '测试',
    colorClass: 'text-violet-700',
    bgClass: 'bg-violet-50',
  },
  tech_lead: {
    icon: CompassIcon,
    label: '技术负责人',
    colorClass: 'text-amber-700',
    bgClass: 'bg-amber-50',
  },
};

function getMemberRoleMeta(role: string) {
  return (
    memberRoleMeta[role] || {
      icon: SearchIcon,
      label: role,
      colorClass: 'text-text',
      bgClass: 'bg-secondary-50',
    }
  );
}

// 缺陷状态分布组件
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
  const maxValue = Math.max(...entries.map(([, v]) => v), 1);

  return (
    <div>
      <div className="text-text-light text-xs mb-3 flex items-center justify-between">
        <span>缺陷状态分布</span>
        <span className="text-text-light">{total} 个</span>
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
              <div className="flex items-center justify-between mb-1">
                <div className="flex items-center gap-2">
                  <span
                    className={`w-6 h-6 rounded-md ${config.bgColor} flex items-center justify-center text-sm`}
                  >
                    <StatusIcon size={14} />
                  </span>
                  <span className="text-sm text-text">{config.label}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`text-sm font-semibold ${config.color}`}>{value}</span>
                  {total > 0 && (
                    <span className="text-xs text-text-light w-10 text-right">
                      {percentage.toFixed(0)}%
                    </span>
                  )}
                </div>
              </div>
              <div className="h-1.5 bg-secondary-100 rounded-full overflow-hidden">
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

// 缺陷严重级别分布组件
function BugSeverityDistribution({ data, total }: { data: Record<string, number>; total: number }) {
  const severityConfig: Record<
    string,
    { label: string; color: string; bgColor: string; borderColor: string; dotColor: string }
  > = {
    critical: {
      label: '严重',
      color: 'text-danger',
      bgColor: 'bg-danger-light',
      borderColor: 'border-danger',
      dotColor: 'bg-danger',
    },
    high: {
      label: '高',
      color: 'text-warning',
      bgColor: 'bg-warning-light',
      borderColor: 'border-warning',
      dotColor: 'bg-warning',
    },
    medium: {
      label: '中',
      color: 'text-info',
      bgColor: 'bg-info-light',
      borderColor: 'border-info',
      dotColor: 'bg-info',
    },
    low: {
      label: '低',
      color: 'text-success',
      bgColor: 'bg-success-light',
      borderColor: 'border-success',
      dotColor: 'bg-success',
    },
  };

  const entries = Object.entries(data).sort((a, b) => {
    const order = ['critical', 'high', 'medium', 'low'];
    return order.indexOf(a[0]) - order.indexOf(b[0]);
  });

  return (
    <div>
      <div className="text-text-light text-xs mb-3">缺陷严重级别分布</div>
      <div className="grid grid-cols-2 gap-2">
        {entries.map(([key, value]) => {
          const config = severityConfig[key] || {
            label: key,
            color: 'text-text',
            bgColor: 'bg-secondary-100',
            borderColor: 'border-border',
            dotColor: 'bg-secondary-400',
          };
          const percentage = total > 0 ? (value / total) * 100 : 0;

          return (
            <div
              key={key}
              className="rounded-lg border border-border bg-white p-3"
            >
              <div className="flex items-start justify-between">
                <span className="inline-flex items-center gap-2">
                  <span className={`h-2.5 w-2.5 rounded-full ${config.dotColor}`} />
                  <span className={`text-sm font-medium ${config.color}`}>{config.label}</span>
                </span>
                <span className={`text-2xl font-bold ${config.color}`}>{value}</span>
              </div>
              {total > 0 && (
                <div className="mt-2 text-xs text-text-light">{percentage.toFixed(1)}%</div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const projectID = Number(id);
  const { user } = useAuthStore();
  const { showSuccess, showError } = useToast();
  const { currentProject, projectOverview, fetchProject, fetchProjectOverview, isLoading, error } =
    useProjectStore();
  const [members, setMembers] = useState<MemberItem[]>([]);
  const [memberError, setMemberError] = useState('');
  const [sprints, setSprints] = useState<SprintSummary[]>([]);
  const [sprintError, setSprintError] = useState('');
  const [selectedSprintID, setSelectedSprintID] = useState<number | null>(null);
  const [burndown, setBurndown] = useState<BurndownReport | null>(null);
  const [burndownError, setBurndownError] = useState('');
  const [isBurndownLoading, setIsBurndownLoading] = useState(false);
  const [velocity, setVelocity] = useState<VelocityReportData | null>(null);
  const [quality, setQuality] = useState<QualityReportData | null>(null);
  const [isReportLoading, setIsReportLoading] = useState(false);
  const [reportError, setReportError] = useState('');
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [selectedMemberUserID, setSelectedMemberUserID] = useState('');
  const [selectedMemberRole, setSelectedMemberRole] = useState<'product' | 'developer' | 'tester'>(
    'developer'
  );
  const [isMemberUpdating, setIsMemberUpdating] = useState(false);
  const [techLeads, setTechLeads] = useState<
    Array<{ id: number; user?: { id: number; email: string } }>
  >([]);
  const [selectedTechLeadUserID, setSelectedTechLeadUserID] = useState('');
  const [isTechLeadUpdating, setIsTechLeadUpdating] = useState(false);
  const [isCreateSprintOpen, setIsCreateSprintOpen] = useState(false);
  const [isSprintSubmitting, setIsSprintSubmitting] = useState(false);
  const [statusUpdatingSprintID, setStatusUpdatingSprintID] = useState<number | null>(null);
  const [confirmAction, setConfirmAction] = useState<ConfirmActionState | null>(null);
  const [sprintForm, setSprintForm] = useState<CreateSprintRequest>({
    name: '',
    goal: '',
    start_date: '',
    end_date: '',
  });
  const [sprintFormError, setSprintFormError] = useState('');
  const project = currentProject?.id === projectID ? currentProject : null;

  useEffect(() => {
    if (!Number.isNaN(projectID) && projectID > 0) {
      fetchProject(projectID);
      fetchProjectOverview(projectID);
      loadMembers(projectID);
      loadSprints(projectID);
      loadReports(projectID);
      loadProjectTechLeads(projectID);
      if (user?.role === 'product' || user?.role === 'admin' || user?.role === 'tech_lead') {
        loadAllUsers();
      }
    }
  }, [projectID, fetchProject, fetchProjectOverview, user?.role]);

  useEffect(() => {
    if (!Number.isNaN(projectID) && projectID > 0 && selectedSprintID) {
      loadBurndown(projectID, selectedSprintID);
    }
  }, [projectID, selectedSprintID]);

  const loadMembers = async (pid: number) => {
    try {
      setMemberError('');
      const data = await projectService.getProjectMembers(pid);
      setMembers(data.members || []);
    } catch {
      setMemberError('成员列表加载失败');
      setMembers([]);
    }
  };

  const loadAllUsers = async () => {
    try {
      const data = await userManagementService.getUsers({ page: 1, limit: 100 });
      setAllUsers(data.users || []);
    } catch {
      setAllUsers([]);
    }
  };

  const loadProjectTechLeads = async (pid: number) => {
    try {
      const data = await techLeadService.getProjectTechLeads(pid);
      setTechLeads(data.tech_leads || []);
    } catch {
      setTechLeads([]);
    }
  };

  const loadSprints = async (pid: number) => {
    try {
      setSprintError('');
      const data = await projectService.getSprints(pid);
      const sprintList = data.sprints || [];
      setSprints(sprintList);
      if (sprintList.length > 0) {
        setSelectedSprintID((prev) => prev || sprintList[0].id);
      } else {
        setSelectedSprintID(null);
        setBurndown(null);
      }
    } catch {
      setSprintError('冲刺列表加载失败');
      setSprints([]);
      setSelectedSprintID(null);
      setBurndown(null);
    }
  };

  const loadBurndown = async (pid: number, sprintID: number) => {
    try {
      setIsBurndownLoading(true);
      setBurndownError('');
      const data = await projectService.getBurndown(pid, sprintID);
      setBurndown(data);
    } catch {
      setBurndown(null);
      setBurndownError('燃尽图数据加载失败');
    } finally {
      setIsBurndownLoading(false);
    }
  };

  const loadReports = async (pid: number) => {
    try {
      setIsReportLoading(true);
      setReportError('');
      const [velocityData, qualityData] = await Promise.all([
        projectService.getVelocity(pid),
        projectService.getQuality(pid),
      ]);
      setVelocity(velocityData);
      setQuality(qualityData);
    } catch (err: unknown) {
      setVelocity(null);
      setQuality(null);
      setReportError(getErrorMessage(err, '报表数据加载失败'));
    } finally {
      setIsReportLoading(false);
    }
  };

  const handleAddMember = async () => {
    const userID = Number(selectedMemberUserID);
    if (!userID) {
      showError('请选择要添加的成员');
      return;
    }
    try {
      setIsMemberUpdating(true);
      await projectService.addProjectMember(projectID, {
        user_id: userID,
        role_in_project: selectedMemberRole,
      });
      showSuccess('项目成员添加成功');
      setSelectedMemberUserID('');
      await loadMembers(projectID);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '成员添加失败'));
    } finally {
      setIsMemberUpdating(false);
    }
  };

  const handleRemoveMember = (member: MemberItem) => {
    if (member.is_owner) {
      showError('项目负责人不能移除');
      return;
    }
    setConfirmAction({
      kind: 'remove_member',
      title: '移除项目成员',
      message: `确认移除成员 ${member.user?.email || member.user_id} 吗？`,
      userID: member.user_id,
    });
  };

  const handleAddTechLead = async () => {
    const userID = Number(selectedTechLeadUserID);
    if (!userID) {
      showError('请选择技术负责人');
      return;
    }
    try {
      setIsTechLeadUpdating(true);
      await techLeadService.addTechLead(projectID, userID);
      showSuccess('技术负责人添加成功');
      setSelectedTechLeadUserID('');
      await loadProjectTechLeads(projectID);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '技术负责人添加失败'));
    } finally {
      setIsTechLeadUpdating(false);
    }
  };

  const handleRemoveTechLead = (userID: number, email?: string) => {
    setConfirmAction({
      kind: 'remove_tech_lead',
      title: '移除技术负责人',
      message: `确认移除技术负责人 ${email || '该成员'} 吗？`,
      userID,
    });
  };

  const handleConfirmAction = async () => {
    if (!confirmAction) {
      return;
    }

    try {
      if (confirmAction.kind === 'remove_member') {
        setIsMemberUpdating(true);
        await projectService.removeProjectMember(projectID, confirmAction.userID);
        showSuccess('成员移除成功');
        await loadMembers(projectID);
      } else {
        setIsTechLeadUpdating(true);
        await techLeadService.removeTechLead(projectID, confirmAction.userID);
        showSuccess('技术负责人移除成功');
        await loadProjectTechLeads(projectID);
      }
      setConfirmAction(null);
    } catch (error: unknown) {
      showError(
        getErrorMessage(
          error,
          confirmAction.kind === 'remove_member' ? '成员移除失败' : '技术负责人移除失败'
        )
      );
    } finally {
      setIsMemberUpdating(false);
      setIsTechLeadUpdating(false);
    }
  };

  const getSprintStatusClass = (status: SprintSummary['status']) => {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-700';
      case 'completed':
        return 'bg-secondary-200 text-text-light';
      default:
        return 'bg-blue-100 text-blue-700';
    }
  };

  const getNextSprintAction = (status: SprintSummary['status']) => {
    if (status === 'planned') {
      return { label: '开始冲刺', target: 'active' as const };
    }
    if (status === 'active') {
      return { label: '完成冲刺', target: 'completed' as const };
    }
    return { label: '重新激活', target: 'active' as const };
  };

  const handleCreateSprint = async () => {
    setSprintFormError('');
    if (!sprintForm.name.trim()) {
      setSprintFormError('请输入冲刺名称');
      return;
    }
    if (!sprintForm.start_date || !sprintForm.end_date) {
      setSprintFormError('请选择开始和结束日期');
      return;
    }
    if (new Date(sprintForm.end_date) < new Date(sprintForm.start_date)) {
      setSprintFormError('结束日期不能早于开始日期');
      return;
    }

    try {
      setIsSprintSubmitting(true);
      await projectService.createSprint(projectID, {
        ...sprintForm,
        name: sprintForm.name.trim(),
        goal: sprintForm.goal?.trim(),
      });
      showSuccess('冲刺创建成功');
      setIsCreateSprintOpen(false);
      setSprintForm({
        name: '',
        goal: '',
        start_date: '',
        end_date: '',
      });
      await loadSprints(projectID);
    } catch (error: unknown) {
      const msg = getErrorMessage(error, '创建冲刺失败');
      setSprintFormError(msg);
      showError(msg);
    } finally {
      setIsSprintSubmitting(false);
    }
  };

  const handleUpdateSprintStatus = async (sprint: SprintSummary) => {
    const action = getNextSprintAction(sprint.status);
    try {
      setStatusUpdatingSprintID(sprint.id);
      await projectService.updateSprintStatus(sprint.id, { status: action.target });
      showSuccess(`冲刺已更新为${formatSprintStatus(action.target)}`);
      await loadSprints(projectID);
      if (selectedSprintID === sprint.id) {
        await loadBurndown(projectID, sprint.id);
      }
    } catch (error: unknown) {
      showError(getErrorMessage(error, '冲刺状态更新失败'));
    } finally {
      setStatusUpdatingSprintID(null);
    }
  };

  if (Number.isNaN(projectID) || projectID <= 0) {
    return <div className="p-8 text-danger">项目ID无效</div>;
  }

  if (isLoading && !project) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center text-text-light">
          <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-primary mx-auto mb-3" />
          <p>加载项目中...</p>
        </div>
      </div>
    );
  }

  if (error && !project) {
    return <div className="p-8 text-danger">{error}</div>;
  }

  const statusBreakdown = projectOverview?.statistics?.status_breakdown || {
    backlog: 0,
    ready: 0,
    in_progress: 0,
    test: 0,
    done: 0,
  };

  const completionRate = projectOverview?.statistics?.completion_rate || 0;
  const totalStories = projectOverview?.statistics?.total_stories || 0;
  const activeMembers = projectOverview?.statistics?.active_members || members.length || 0;
  const inProgressStories = statusBreakdown.in_progress || 0;
  const projectModeLabel = project?.agile_mode === 'scrum' ? '冲刺模式' : '看板模式';
  const canManageMembers = user?.role === 'product' || user?.role === 'admin';
  const canManageTechLeads = user?.role === 'admin';
  const canCreateStory = user?.role === 'product' || user?.role === 'admin';
  const availableMemberUsers = allUsers.filter(
    (candidate) => !members.some((member) => member.user_id === candidate.id)
  );
  const availableTechLeadUsers = allUsers.filter(
    (candidate) =>
      candidate.role === 'tech_lead' &&
      !techLeads.some((techLead) => techLead.user && techLead.user.id === candidate.id)
  );

  return (
    <div className="space-y-6 px-4 pb-6 sm:px-6 lg:px-8">
      <section className="surface-card overflow-hidden rounded-[2rem]">
        <div className="space-y-6 px-5 py-6 sm:px-6 lg:px-8 lg:py-8">
          <div className="space-y-5">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary">
                  {project?.agile_mode === 'scrum' ? <SprintIcon size={14} /> : <BoardIcon size={14} />}
                  {projectModeLabel}
                </span>
                {project?.owner && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-700">
                    <CrownIcon size={12} />
                    负责人 {project.owner.email}
                  </span>
                )}
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-text sm:text-4xl">
                  {project?.name || '项目详情'}
                </h1>
                <p className="mt-2 max-w-3xl text-sm leading-6 text-text-light sm:text-base">
                  {project?.description || '暂无项目描述。这里查看项目节奏、质量、成员和冲刺进展。'}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">总故事数</div>
                <div className="mt-2 inline-flex items-center gap-2 text-2xl font-semibold text-text">
                  <StoryIcon size={20} />
                  {totalStories}
                </div>
              </div>
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">推进中</div>
                <div className="mt-2 inline-flex items-center gap-2 text-2xl font-semibold text-primary">
                  <WrenchIcon size={20} />
                  {inProgressStories}
                </div>
              </div>
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">完成率</div>
                <div className="mt-2 inline-flex items-center gap-2 text-2xl font-semibold text-success">
                  <CheckCircleIcon size={20} />
                  {completionRate.toFixed(1)}%
                </div>
              </div>
              <div className="rounded-2xl border border-border bg-white p-4">
                <div className="text-xs font-medium text-text-light">活跃成员</div>
                <div className="mt-2 inline-flex items-center gap-2 text-2xl font-semibold text-text">
                  <UsersIcon size={20} />
                  {activeMembers}
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-[1.6rem] border border-border bg-secondary-50 p-4 sm:p-5">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <h2 className="text-base font-semibold text-text sm:text-lg">工作入口</h2>
                <p className="mt-1 text-sm text-text-light">
                  从这里继续进入看板、缺陷或故事创建，首屏统计只负责说明项目状态。
                </p>
              </div>
              <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:justify-end">
                <Link to={`/projects/${projectID}/board`}>
                  <Button className="w-full sm:w-auto">进入看板</Button>
                </Link>
                <Link to={`/projects/${projectID}/bugs`}>
                  <Button variant="secondary" className="w-full sm:w-auto">
                    缺陷管理
                  </Button>
                </Link>
                {canCreateStory ? (
                  <Link to={`/projects/${projectID}/stories/new`}>
                    <Button variant="secondary" className="w-full sm:w-auto">
                      创建故事
                    </Button>
                  </Link>
                ) : (
                  <div className="rounded-xl border border-border bg-white px-3 py-2 text-xs text-text-light">
                    当前角色无创建故事权限
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </section>

        <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-text">冲刺管理</h2>
              <p className="mt-1 text-sm text-text-light">统一查看每个冲刺的周期、完成量和下一步动作。</p>
            </div>
            <Button size="sm" onClick={() => setIsCreateSprintOpen(true)}>
              + 新建冲刺
          </Button>
        </div>

        {sprintError && <div className="state-panel state-panel-error mb-3">{sprintError}</div>}
        {sprints.length === 0 ? (
          <div className="state-panel state-panel-empty">暂无冲刺，先创建一个冲刺</div>
        ) : (
          <div className="space-y-3">
            {sprints.map((sprint) => {
              const action = getNextSprintAction(sprint.status);
              return (
                <div
                  key={sprint.id}
                  className="flex flex-col gap-3 rounded-lg border border-border px-4 py-3 md:flex-row md:items-center md:justify-between"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium text-text truncate">{sprint.name}</span>
                      <span
                        className={`text-xs px-2 py-0.5 rounded-full ${getSprintStatusClass(sprint.status)}`}
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
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => setSelectedSprintID(sprint.id)}
                    >
                      查看燃尽图
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => handleUpdateSprintStatus(sprint)}
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

      <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
        <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-text">燃尽图</h2>
            <p className="mt-1 text-sm text-text-light">把理想线和实际线放在一起看，快速判断冲刺节奏是否健康。</p>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <span className="text-sm text-text-light">冲刺</span>
            <select
              value={selectedSprintID || ''}
              onChange={(e) => setSelectedSprintID(e.target.value ? Number(e.target.value) : null)}
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
                  loadBurndown(projectID, selectedSprintID);
                }
              }}
            >
              刷新
            </Button>
          </div>
        </div>

        {sprintError && <div className="state-panel state-panel-error mb-3">{sprintError}</div>}
        {isBurndownLoading && (
          <div className="state-panel state-panel-loading">燃尽图加载中...</div>
        )}
        {!isBurndownLoading && burndownError && (
          <div className="state-panel state-panel-error">{burndownError}</div>
        )}
        {!isBurndownLoading && !burndownError && burndown && <BurndownChart report={burndown} />}
        {!isBurndownLoading && !burndown && !burndownError && (
          <div className="state-panel state-panel-empty">请选择冲刺查看燃尽图</div>
        )}
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-text">速度报表</h2>
              <p className="mt-1 text-sm text-text-light">用已完成点数和计划点数判断团队交付节奏。</p>
            </div>
            <Button size="sm" variant="secondary" onClick={() => loadReports(projectID)}>
              刷新
            </Button>
          </div>
          {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
          {!isReportLoading && reportError && (
            <div className="state-panel state-panel-error">{reportError}</div>
          )}
          {!isReportLoading && !reportError && (!velocity || velocity.velocity.length === 0) && (
            <div className="state-panel state-panel-empty">暂无冲刺速度数据</div>
          )}
          {!isReportLoading && !reportError && velocity && velocity.velocity.length > 0 && (
            <div className="space-y-2">
              {velocity.velocity.map((item) => (
                <div key={item.sprint_id} className="border border-border rounded-lg p-3">
                  <div className="font-medium text-text">{item.name}</div>
                  <div className="text-xs text-text-light mt-1">
                    状态：{formatSprintStatus(item.status)} · 完成点数 {item.completed_points}/{item.planned_points} ·
                    速度 {item.velocity.toFixed(1)}%
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
          <div className="mb-4">
            <h2 className="text-lg font-semibold text-text">质量报表</h2>
            <p className="mt-1 text-sm text-text-light">结合缺陷和 AC 完成率看当前质量风险。</p>
          </div>
          {isReportLoading && <div className="state-panel state-panel-loading">报表加载中...</div>}
          {!isReportLoading && reportError && (
            <div className="state-panel state-panel-error">{reportError}</div>
          )}
          {!isReportLoading && !reportError && quality && (
            <div className="space-y-4 text-sm">
              <div className="grid grid-cols-2 gap-3">
                <div className="bg-secondary-50 rounded-lg p-3">
                  <div className="text-text-light text-xs">缺陷总数</div>
                  <div className="text-xl font-semibold text-text mt-1">{quality.bugs.total}</div>
                </div>
                <div className="bg-secondary-50 rounded-lg p-3">
                  <div className="text-text-light text-xs">AC 完成率</div>
                  <div className="text-xl font-semibold text-text mt-1">
                    {quality.acceptance_criteria.completion_percentage.toFixed(1)}%
                  </div>
                </div>
              </div>

              {/* 缺陷状态分布 */}
              <BugStatusDistribution
                data={quality.bugs.status_breakdown}
                total={quality.bugs.total}
              />

              {/* 缺陷严重级别分布 */}
              <BugSeverityDistribution
                data={quality.bugs.severity_breakdown}
                total={quality.bugs.total}
              />
            </div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
          <div className="mb-4">
            <h2 className="text-lg font-semibold text-text">状态分布</h2>
            <p className="mt-1 text-sm text-text-light">快速看当前故事主要积压在哪个阶段。</p>
          </div>
          <div className="space-y-3">
            {Object.entries(statusBreakdown).map(([status, count]) => (
              <div key={status} className="flex items-center justify-between">
                <span className="text-text-light">{formatStoryStatus(status)}</span>
                <span className="font-medium text-text">{count}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
          <div className="mb-4 flex flex-col gap-2">
            <h2 className="text-lg font-semibold text-text">项目成员</h2>
            <div className="flex flex-wrap items-center gap-2 text-xs text-text-light">
              <span className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-1 text-blue-700">
                <ClipboardIcon size={12} />
                产品
              </span>
              <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-1 text-emerald-700">
                <CodeIcon size={12} />
                开发
              </span>
              <span className="inline-flex items-center gap-1 rounded-full bg-violet-50 px-2 py-1 text-violet-700">
                <SearchIcon size={12} />
                测试
              </span>
              <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2 py-1 text-amber-700">
                <CrownIcon size={12} />
                负责人
              </span>
            </div>
          </div>
          {canManageMembers && (
            <div className="mb-4 grid grid-cols-1 gap-2 md:grid-cols-3">
              <select
                value={selectedMemberUserID}
                onChange={(e) => setSelectedMemberUserID(e.target.value)}
                className="field-control"
              >
                <option value="">选择用户</option>
                {availableMemberUsers.map((candidate) => (
                  <option key={candidate.id} value={candidate.id}>
                    {candidate.email}（{candidate.role}）
                  </option>
                ))}
              </select>
              <select
                value={selectedMemberRole}
                onChange={(e) =>
                  setSelectedMemberRole(e.target.value as 'product' | 'developer' | 'tester')
                }
                className="field-control"
              >
                <option value="product">产品经理</option>
                <option value="developer">开发</option>
                <option value="tester">测试</option>
              </select>
              <Button size="sm" onClick={handleAddMember} isLoading={isMemberUpdating}>
                添加成员
              </Button>
            </div>
          )}
          {memberError ? (
            <div className="state-panel state-panel-error">{memberError}</div>
          ) : members.length === 0 ? (
            <div className="state-panel state-panel-empty">暂无成员</div>
          ) : (
            <div className="space-y-3">
              {members.map((member) => {
                const roleMeta = getMemberRoleMeta(member.role_in_project);
                const RoleIcon = roleMeta.icon;
                return (
                  <div
                    key={member.id}
                    className="flex flex-col gap-3 rounded-lg border border-border px-3 py-3 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-sm font-medium text-primary">
                          {(member.user?.email || `#${member.user_id}`).charAt(0).toUpperCase()}
                        </div>
                        <div className="min-w-0">
                          <div className="truncate text-sm font-medium text-text">
                            {member.user?.email || `用户 #${member.user_id}`}
                          </div>
                          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-text-light">
                            <span
                              className={`inline-flex items-center rounded-full px-2 py-1 ${roleMeta.bgClass} ${roleMeta.colorClass}`}
                              title={roleMeta.label}
                              aria-label={roleMeta.label}
                            >
                              <RoleIcon size={12} />
                            </span>
                            {member.is_owner && (
                              <span
                                className="inline-flex items-center rounded-full bg-amber-50 px-2 py-1 text-amber-700"
                                title="项目负责人"
                                aria-label="项目负责人"
                              >
                                <CrownIcon size={12} />
                              </span>
                            )}
                            <span>{new Date(member.joined_at).toLocaleDateString()}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center justify-end gap-2">
                      {canManageMembers && !member.is_owner && (
                        <Button
                          size="sm"
                          variant="danger"
                          onClick={() => handleRemoveMember(member)}
                          isLoading={isMemberUpdating}
                        >
                          移除
                        </Button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      <div className="surface-card rounded-[1.8rem] p-5 sm:p-6">
          <div className="mb-4">
            <h2 className="text-lg font-semibold text-text">项目技术负责人</h2>
            <p className="mt-1 text-sm text-text-light">明确技术把关角色，减少决策链路里的模糊地带。</p>
          </div>
        {canManageTechLeads && (
          <div className="mb-4 grid grid-cols-1 gap-2 md:grid-cols-2">
            <select
              value={selectedTechLeadUserID}
              onChange={(e) => setSelectedTechLeadUserID(e.target.value)}
              className="field-control"
            >
              <option value="">选择技术负责人</option>
              {availableTechLeadUsers.map((candidate) => (
                <option key={candidate.id} value={candidate.id}>
                  {candidate.email}
                </option>
              ))}
            </select>
            <Button size="sm" onClick={handleAddTechLead} isLoading={isTechLeadUpdating}>
              添加技术负责人
            </Button>
          </div>
        )}
        {techLeads.length === 0 ? (
          <div className="state-panel state-panel-empty">当前项目暂无技术负责人</div>
        ) : (
          <div className="space-y-2">
            {techLeads.map((item) => (
              <div
                key={item.id}
                className="flex flex-col gap-3 rounded-lg border border-border px-3 py-3 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex items-center gap-2">
                  <div className="inline-flex items-center rounded-full bg-amber-50 px-2 py-1 text-amber-700">
                    <CompassIcon size={12} />
                  </div>
                  <div className="text-sm text-text">{item.user?.email || '未知用户'}</div>
                </div>
                {canManageTechLeads && item.user && (
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() => handleRemoveTechLead(item.user!.id, item.user?.email)}
                    isLoading={isTechLeadUpdating}
                  >
                    移除
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <Modal
        isOpen={Boolean(confirmAction)}
        onClose={() => {
          if (!isMemberUpdating && !isTechLeadUpdating) {
            setConfirmAction(null);
          }
        }}
        title={confirmAction?.title}
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-text">{confirmAction?.message}</p>
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setConfirmAction(null)}
              disabled={isMemberUpdating || isTechLeadUpdating}
            >
              取消
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmAction}
              isLoading={isMemberUpdating || isTechLeadUpdating}
            >
              确认移除
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={isCreateSprintOpen}
        onClose={() => {
          if (!isSprintSubmitting) {
            setIsCreateSprintOpen(false);
            setSprintFormError('');
          }
        }}
        title="新建冲刺"
        size="md"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-text mb-2">冲刺名称</label>
            <input
              type="text"
              value={sprintForm.name}
              onChange={(e) => setSprintForm((prev) => ({ ...prev, name: e.target.value }))}
              className="field-control"
              placeholder="例如：Sprint 1"
              maxLength={120}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text mb-2">目标（可选）</label>
            <textarea
              value={sprintForm.goal}
              onChange={(e) => setSprintForm((prev) => ({ ...prev, goal: e.target.value }))}
              className="field-control"
              rows={3}
              maxLength={500}
              placeholder="本次冲刺要达成什么"
            />
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-text mb-2">开始日期</label>
              <input
                type="date"
                value={sprintForm.start_date}
                onChange={(e) => setSprintForm((prev) => ({ ...prev, start_date: e.target.value }))}
                className="field-control"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text mb-2">结束日期</label>
              <input
                type="date"
                value={sprintForm.end_date}
                onChange={(e) => setSprintForm((prev) => ({ ...prev, end_date: e.target.value }))}
                className="field-control"
              />
            </div>
          </div>
          {sprintFormError && (
            <div className="state-panel state-panel-error">{sprintFormError}</div>
          )}
          <div className="flex flex-col-reverse gap-3 pt-2 sm:flex-row sm:justify-end">
            <Button
              variant="secondary"
              onClick={() => setIsCreateSprintOpen(false)}
              disabled={isSprintSubmitting}
            >
              取消
            </Button>
            <Button onClick={handleCreateSprint} isLoading={isSprintSubmitting}>
              创建冲刺
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

function BurndownChart({ report }: { report: BurndownReport }) {
  const points = report.points || [];
  if (points.length === 0 || report.baseline_points <= 0) {
    return (
      <div className="state-panel state-panel-empty space-y-2">
        <div>当前冲刺暂无可燃尽的数据</div>
        <div>
          请先把故事规划到该冲刺，并设置故事点；完成故事后实际线才会下降。
        </div>
      </div>
    );
  }

  const width = 900;
  const height = 280;
  const paddingX = 40;
  const paddingY = 24;
  const plotWidth = width - paddingX * 2;
  const plotHeight = height - paddingY * 2;
  const maxY = Math.max(report.baseline_points, ...points.map((p) => p.remaining_points), 1);
  const xDivisor = Math.max(points.length - 1, 1);
  const dayMS = 24 * 60 * 60 * 1000;

  const parseDateOnly = (raw: string) => {
    const datePart = raw.slice(0, 10);
    const [year, month, day] = datePart.split('-').map((n) => Number(n));
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
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-3 text-sm">
        <div className="bg-secondary-50 border border-border rounded-lg px-3 py-2">
          <div className="text-text-light text-xs">基线点数</div>
          <div className="font-semibold text-text">{report.baseline_points}</div>
        </div>
        <div className="bg-secondary-50 border border-border rounded-lg px-3 py-2">
          <div className="text-text-light text-xs">当前剩余（实际）</div>
          <div className="font-semibold text-text">{currentRemaining}</div>
        </div>
        <div className="bg-secondary-50 border border-border rounded-lg px-3 py-2">
          <div className="text-text-light text-xs">今日理想剩余</div>
          <div className="font-semibold text-text">{idealRemaining}</div>
        </div>
      </div>

      <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-64">
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

      <div className="flex flex-wrap items-center justify-between gap-3 mt-3 text-xs text-text-light">
        <div className="flex items-center gap-4">
          <span className="inline-flex items-center gap-1">
            <span className="w-4 h-[2px] bg-slate-400 inline-block" />
            理想线
          </span>
          <span className="inline-flex items-center gap-1">
            <span className="w-4 h-[2px] bg-primary inline-block" />
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

      <div className="mt-3 text-xs text-text-light space-y-1">
        <div>理想线：基线点数在冲刺总天数内按线性下降计算，不会因为“今天”提前归零。</div>
        <div>实际线：到每天结束时，状态为“已完成”的故事点从基线中扣减后的剩余点数。</div>
        <div>统计范围：仅统计已规划到当前冲刺的故事。</div>
      </div>
    </div>
  );
}
