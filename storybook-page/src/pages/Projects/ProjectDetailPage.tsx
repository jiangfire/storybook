import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useProjectStore } from '../../stores/projectStore';
import { useAuthStore } from '../../stores/authStore';
import { projectService } from '../../services/projectService';
import { techLeadService } from '../../services/techLeadService';
import { userManagementService } from '../../services/userManagementService';
import { useToast } from '../../components/ui/Toast';
import { PageContainer, PageHero } from '../../components/page/PageLayout';
import { formatSprintStatus } from '../../utils/formatters';
import { getErrorMessage } from '../../utils/error';
import {
  canCreateStory as canCreateStoryPermission,
  canManageProjectMembers,
  canManageTechLeads as canManageTechLeadsPermission,
} from '../../utils/permissions';
import type {
  BurndownReport,
  CreateSprintRequest,
  QualityReportData,
  SprintSummary,
  VelocityReportData,
} from '../../types/api';
import type { ProjectRole, User } from '../../types/models';
import { BurndownSection } from './projectDetail/BurndownSection';
import { ProjectHeroSection } from './projectDetail/ProjectHeroSection';
import { ProjectMembersSection } from './projectDetail/ProjectMembersSection';
import { ProjectStatusSection } from './projectDetail/ProjectStatusSection';
import { ProjectTechLeadsSection } from './projectDetail/ProjectTechLeadsSection';
import { ReportsSection } from './projectDetail/ReportsSection';
import { SprintManagementSection } from './projectDetail/SprintManagementSection';
import { ConfirmActionModal } from './projectDetail/ConfirmActionModal';
import { SprintCreateModal } from './projectDetail/SprintCreateModal';
import { getNextSprintAction } from './projectDetail/sprintHelpers';
import type { ConfirmActionState, ProjectMemberItem } from './projectDetail/types';

const emptySprintForm: CreateSprintRequest = {
  name: '',
  goal: '',
  start_date: '',
  end_date: '',
};

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const projectID = Number(id);
  const { user } = useAuthStore();
  const { showSuccess, showError } = useToast();
  const { currentProject, projectOverview, fetchProject, fetchProjectOverview, isLoading, error } =
    useProjectStore();
  const [members, setMembers] = useState<ProjectMemberItem[]>([]);
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
  const [memberCandidates, setMemberCandidates] = useState<User[]>([]);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [selectedMemberUserID, setSelectedMemberUserID] = useState('');
  const [selectedMemberRole, setSelectedMemberRole] = useState<ProjectRole>('developer');
  const [isAddingMember, setIsAddingMember] = useState(false);
  const [removingMemberUserID, setRemovingMemberUserID] = useState<number | null>(null);
  const [techLeads, setTechLeads] = useState<User[]>([]);
  const [selectedTechLeadUserID, setSelectedTechLeadUserID] = useState('');
  const [isAddingTechLead, setIsAddingTechLead] = useState(false);
  const [removingTechLeadID, setRemovingTechLeadID] = useState<number | null>(null);
  const [isCreateSprintOpen, setIsCreateSprintOpen] = useState(false);
  const [isSprintSubmitting, setIsSprintSubmitting] = useState(false);
  const [statusUpdatingSprintID, setStatusUpdatingSprintID] = useState<number | null>(null);
  const [confirmAction, setConfirmAction] = useState<ConfirmActionState | null>(null);
  const [sprintForm, setSprintForm] = useState<CreateSprintRequest>(emptySprintForm);
  const [sprintFormError, setSprintFormError] = useState('');
  const project = currentProject?.id === projectID ? currentProject : null;
  const canManageMembers = canManageProjectMembers(project);
  const canManageTechLeads = canManageTechLeadsPermission(user?.role);
  const canCreateStory = canCreateStoryPermission(user?.role);

  const loadMembers = useCallback(async (pid: number) => {
    try {
      setMemberError('');
      const data = await projectService.getProjectMembers(pid);
      const memberList: ProjectMemberItem[] = data.members || [];
      setMembers(memberList);
    } catch {
      setMemberError('成员列表加载失败');
      setMembers([]);
    }
  }, []);

  const loadMemberCandidates = useCallback(async (pid: number) => {
    try {
      const data = await projectService.getProjectMemberCandidates(pid);
      setMemberCandidates(data.users || []);
    } catch {
      setMemberCandidates([]);
    }
  }, []);

  const loadAllUsers = useCallback(async () => {
    try {
      const data = await userManagementService.getUsers({ page: 1, limit: 100 });
      setAllUsers(data.users || []);
    } catch {
      setAllUsers([]);
    }
  }, []);

  const loadProjectTechLeads = useCallback(async (pid: number) => {
    try {
      const data = await techLeadService.getProjectTechLeads(pid);
      setTechLeads(data.tech_leads || []);
    } catch {
      setTechLeads([]);
    }
  }, []);

  const loadSprints = useCallback(async (pid: number, preferredSprintID: number | null) => {
    try {
      setSprintError('');
      const data = await projectService.getSprints(pid);
      const sprintList = data.sprints || [];
      setSprints(sprintList);
      if (sprintList.length > 0) {
        const nextSelectedSprintID =
          preferredSprintID && sprintList.some((sprint) => sprint.id === preferredSprintID)
            ? preferredSprintID
            : sprintList[0].id;
        setSelectedSprintID(nextSelectedSprintID);
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
  }, []);

  const loadBurndown = useCallback(async (pid: number, sprintID: number) => {
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
  }, []);

  const loadReports = useCallback(async (pid: number) => {
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
  }, []);

  useEffect(() => {
    if (!Number.isNaN(projectID) && projectID > 0) {
      setSelectedSprintID(null);
      setBurndown(null);
      setBurndownError('');
    }
  }, [projectID]);

  useEffect(() => {
    if (!Number.isNaN(projectID) && projectID > 0) {
      fetchProject(projectID);
      fetchProjectOverview(projectID);
      void loadMembers(projectID);
      void loadSprints(projectID, null);
      void loadReports(projectID);
      void loadProjectTechLeads(projectID);
    }
  }, [
    projectID,
    fetchProject,
    fetchProjectOverview,
    loadMembers,
    loadProjectTechLeads,
    loadReports,
    loadSprints,
  ]);

  useEffect(() => {
    if (Number.isNaN(projectID) || projectID <= 0) {
      return;
    }
    if (canManageMembers) {
      void loadMemberCandidates(projectID);
    } else {
      setMemberCandidates([]);
    }
  }, [projectID, canManageMembers, loadMemberCandidates]);

  useEffect(() => {
    if (Number.isNaN(projectID) || projectID <= 0) {
      return;
    }
    if (canManageTechLeads) {
      void loadAllUsers();
    } else {
      setAllUsers([]);
    }
  }, [projectID, canManageTechLeads, loadAllUsers]);

  useEffect(() => {
    if (!Number.isNaN(projectID) && projectID > 0 && selectedSprintID) {
      void loadBurndown(projectID, selectedSprintID);
    }
  }, [projectID, selectedSprintID, loadBurndown]);

  const handleAddMember = async () => {
    const userID = Number(selectedMemberUserID);
    if (!userID) {
      showError('请选择要添加的成员');
      return;
    }
    try {
      setIsAddingMember(true);
      await projectService.addProjectMember(projectID, {
        user_id: userID,
        role_in_project: selectedMemberRole,
      });
      showSuccess('项目成员添加成功');
      setSelectedMemberUserID('');
      await loadMembers(projectID);
      await loadMemberCandidates(projectID);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '成员添加失败'));
    } finally {
      setIsAddingMember(false);
    }
  };

  const handleRemoveMember = (member: ProjectMemberItem) => {
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
      setIsAddingTechLead(true);
      await techLeadService.addTechLead(projectID, userID);
      showSuccess('技术负责人添加成功');
      setSelectedTechLeadUserID('');
      await loadProjectTechLeads(projectID);
    } catch (error: unknown) {
      showError(getErrorMessage(error, '技术负责人添加失败'));
    } finally {
      setIsAddingTechLead(false);
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
        setRemovingMemberUserID(confirmAction.userID);
        await projectService.removeProjectMember(projectID, confirmAction.userID);
        showSuccess('成员移除成功');
        await loadMembers(projectID);
        await loadMemberCandidates(projectID);
      } else {
        setRemovingTechLeadID(confirmAction.userID);
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
      setRemovingMemberUserID(null);
      setRemovingTechLeadID(null);
    }
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
      setSprintForm(emptySprintForm);
      await loadSprints(projectID, selectedSprintID);
    } catch (error: unknown) {
      const msg = getErrorMessage(error, '创建冲刺失败');
      setSprintFormError(msg);
      showError(msg);
    } finally {
      setIsSprintSubmitting(false);
    }
  };

  const isConfirmSubmitting = Boolean(removingMemberUserID || removingTechLeadID);
  const handleCloseConfirmModal = () => {
    if (!isConfirmSubmitting) {
      setConfirmAction(null);
    }
  };
  const handleSprintFormChange = (field: keyof CreateSprintRequest, value: string) => {
    setSprintForm((prev) => ({ ...prev, [field]: value }));
  };
  const handleCloseSprintModal = () => {
    if (!isSprintSubmitting) {
      setIsCreateSprintOpen(false);
      setSprintFormError('');
    }
  };

  const handleUpdateSprintStatus = async (sprint: SprintSummary) => {
    const action = getNextSprintAction(sprint.status);
    try {
      setStatusUpdatingSprintID(sprint.id);
      await projectService.updateSprintStatus(sprint.id, { status: action.target });
      showSuccess(`冲刺已更新为${formatSprintStatus(action.target)}`);
      await loadSprints(projectID, selectedSprintID);
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
    pending: 0,
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
  const availableMemberUsers = memberCandidates.filter(
    (candidate) => !members.some((member) => member.user_id === candidate.id)
  );
  const availableTechLeadUsers = allUsers.filter(
    (candidate) =>
      candidate.role === 'tech_lead' &&
      !techLeads.some((techLead) => techLead.id === candidate.id)
  );

  return (
    <PageContainer>
      <PageHero className="border-primary-100 bg-gradient-to-br from-white via-secondary-50 to-primary-50/60 shadow-[0_20px_44px_-38px_rgba(16,42,67,0.28)]">
        <ProjectHeroSection
          project={project}
          projectID={projectID}
          projectModeLabel={projectModeLabel}
          totalStories={totalStories}
          inProgressStories={inProgressStories}
          completionRate={completionRate}
          activeMembers={activeMembers}
          canCreateStory={canCreateStory}
        />
      </PageHero>

      <SprintManagementSection
        sprintError={sprintError}
        sprints={sprints}
        statusUpdatingSprintID={statusUpdatingSprintID}
        onCreateSprint={() => setIsCreateSprintOpen(true)}
        onSelectSprint={setSelectedSprintID}
        onUpdateSprintStatus={handleUpdateSprintStatus}
      />

      <BurndownSection
        projectID={projectID}
        sprintError={sprintError}
        sprints={sprints}
        selectedSprintID={selectedSprintID}
        isBurndownLoading={isBurndownLoading}
        burndownError={burndownError}
        burndown={burndown}
        onSelectSprint={setSelectedSprintID}
        onRefresh={loadBurndown}
      />

      <ReportsSection
        isReportLoading={isReportLoading}
        reportError={reportError}
        velocity={velocity}
        quality={quality}
        onRefresh={() => void loadReports(projectID)}
      />

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <ProjectStatusSection statusBreakdown={statusBreakdown} />
        <ProjectMembersSection
          canManageMembers={canManageMembers}
          memberError={memberError}
          members={members}
          availableMemberUsers={availableMemberUsers}
          selectedMemberUserID={selectedMemberUserID}
          selectedMemberRole={selectedMemberRole}
          isAddingMember={isAddingMember}
          removingMemberUserID={removingMemberUserID}
          onSelectedMemberUserIDChange={setSelectedMemberUserID}
          onSelectedMemberRoleChange={setSelectedMemberRole}
          onAddMember={() => void handleAddMember()}
          onRemoveMember={handleRemoveMember}
        />
      </div>

      <ProjectTechLeadsSection
        canManageTechLeads={canManageTechLeads}
        techLeads={techLeads}
        availableTechLeadUsers={availableTechLeadUsers}
        selectedTechLeadUserID={selectedTechLeadUserID}
        isAddingTechLead={isAddingTechLead}
        removingTechLeadID={removingTechLeadID}
        onSelectedTechLeadUserIDChange={setSelectedTechLeadUserID}
        onAddTechLead={() => void handleAddTechLead()}
        onRemoveTechLead={handleRemoveTechLead}
      />

      <ConfirmActionModal
        confirmAction={confirmAction}
        isSubmitting={isConfirmSubmitting}
        onCancel={handleCloseConfirmModal}
        onConfirm={handleConfirmAction}
      />

      <SprintCreateModal
        isOpen={isCreateSprintOpen}
        sprintForm={sprintForm}
        sprintFormError={sprintFormError}
        isSubmitting={isSprintSubmitting}
        onClose={handleCloseSprintModal}
        onChange={handleSprintFormChange}
        onSubmit={handleCreateSprint}
      />
    </PageContainer>
  );
}
