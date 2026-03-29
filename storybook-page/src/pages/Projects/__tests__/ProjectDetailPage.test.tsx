import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { ProjectOverview, User } from '../../../types/models';
import type {
  BurndownReport,
  QualityReportData,
  SprintSummary,
  VelocityReportData,
} from '../../../types/api';
import { useAuthStore } from '../../../stores/authStore';
import { useProjectStore } from '../../../stores/projectStore';
import { projectService } from '../../../services/projectService';
import { techLeadService } from '../../../services/techLeadService';
import { userManagementService } from '../../../services/userManagementService';
import ProjectDetailPage from '../ProjectDetailPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/projectService', () => ({
  projectService: {
    getProjectMembers: vi.fn(),
    getProjectMemberCandidates: vi.fn(),
    addProjectMember: vi.fn(),
    removeProjectMember: vi.fn(),
    getSprints: vi.fn(),
    createSprint: vi.fn(),
    updateSprintStatus: vi.fn(),
    getBurndown: vi.fn(),
    getVelocity: vi.fn(),
    getQuality: vi.fn(),
  },
}));

vi.mock('../../../services/techLeadService', () => ({
  techLeadService: {
    getProjectTechLeads: vi.fn(),
    addTechLead: vi.fn(),
    removeTechLead: vi.fn(),
  },
}));

vi.mock('../../../services/userManagementService', () => ({
  userManagementService: {
    getUsers: vi.fn(),
  },
}));

const mockedProjectService = vi.mocked(projectService, { deep: true });
const mockedTechLeadService = vi.mocked(techLeadService, { deep: true });
const mockedUserManagementService = vi.mocked(userManagementService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

const ownerUser: User = {
  id: 1,
  email: 'owner@example.com',
  role: 'product',
  created_at: NOW,
};

const candidateUser: User = {
  id: 3,
  email: 'newdev@example.com',
  role: 'developer',
  created_at: NOW,
};

const techLeadUser: User = {
  id: 4,
  email: 'techlead@example.com',
  role: 'tech_lead',
  created_at: NOW,
};

const anotherDeveloper: User = {
  id: 5,
  email: 'otherdev@example.com',
  role: 'developer',
  created_at: NOW,
};

const projectOverview: ProjectOverview = {
  project: {
    id: 1,
    name: 'Alpha',
  },
  statistics: {
    total_stories: 3,
    status_breakdown: {
      pending: 1,
      backlog: 1,
      ready: 0,
      in_progress: 1,
      test: 0,
      done: 0,
    },
    completion_rate: 33.3,
    active_members: 1,
    avg_story_points: 5,
  },
  recent_activities: [],
};

const velocityData: VelocityReportData = {
  project_id: 1,
  velocity: [],
};

const qualityData: QualityReportData = {
  project_id: 1,
  bugs: {
    total: 0,
    status_breakdown: {},
    severity_breakdown: {},
  },
  acceptance_criteria: {
    total: 0,
    passed: 0,
    failed: 0,
    completion_percentage: 0,
  },
};

const richVelocityData: VelocityReportData = {
  project_id: 1,
  velocity: [
    {
      sprint_id: 31,
      name: 'Sprint 1',
      status: 'active',
      start_date: '2026-04-01',
      end_date: '2026-04-14',
      story_count: 5,
      planned_points: 20,
      completed_points: 15,
      velocity: 75,
    },
  ],
};

const richQualityData: QualityReportData = {
  project_id: 1,
  bugs: {
    total: 5,
    status_breakdown: {
      resolved: 3,
      open: 2,
    },
    severity_breakdown: {
      critical: 1,
      high: 2,
      medium: 2,
    },
  },
  acceptance_criteria: {
    total: 10,
    passed: 7,
    failed: 3,
    completion_percentage: 70,
  },
};

const activeSprint: SprintSummary = {
  id: 31,
  project_id: 1,
  name: 'Sprint 1',
  goal: '完成核心登录链路',
  start_date: '2026-04-01',
  end_date: '2026-04-14',
  status: 'active',
  total_stories: 5,
  done_stories: 3,
  created_at: NOW,
  updated_at: NOW,
};

const completedSprint: SprintSummary = {
  ...activeSprint,
  status: 'completed',
  done_stories: 5,
};

const burndownData: BurndownReport = {
  project_id: 1,
  sprint: {
    id: 31,
    name: 'Sprint 1',
    start_date: '2026-04-01',
    end_date: '2026-04-14',
  },
  baseline_points: 20,
  points: [
    { date: '2026-04-01', remaining_points: 20 },
    { date: '2026-04-07', remaining_points: 12 },
    { date: '2026-04-14', remaining_points: 8 },
  ],
};

function setAuthUser(role: User['role'] = 'product') {
  useAuthStore.setState({
    user: {
      id: 1,
      email: 'owner@example.com',
      role,
      created_at: NOW,
    },
    token: 'token',
    isAuthenticated: true,
    isLoading: false,
    error: null,
  });
}

function setProjectStore(isOwner: boolean) {
  useProjectStore.setState({
    currentProject: {
      id: 1,
      name: 'Alpha',
      description: '核心项目',
      agile_mode: 'kanban',
      owner: ownerUser,
      is_owner: isOwner,
      created_at: NOW,
      updated_at: NOW,
    },
    projectOverview,
    projects: [],
    isLoading: false,
    error: null,
    fetchProjects: vi.fn(async () => {}),
    fetchProject: vi.fn(async () => {}),
    fetchProjectOverview: vi.fn(async () => {}),
    createProject: vi.fn(async () => {
      throw new Error('not implemented');
    }),
    updateProject: vi.fn(async () => {}),
    deleteProject: vi.fn(async () => {}),
    setCurrentProject: vi.fn(),
    clearError: vi.fn(),
  });
}

function getSelectByDefaultOption(optionName: string) {
  const select = screen
    .getAllByRole('combobox')
    .find((element) => within(element).queryByRole('option', { name: optionName }));

  if (!(select instanceof HTMLSelectElement)) {
    throw new Error(`select with option "${optionName}" not found`);
  }

  return select;
}

function getSectionByHeading(name: string) {
  const heading = screen.getByRole('heading', { name });
  const section = heading.closest('.section-card');
  if (!(section instanceof HTMLElement)) {
    throw new Error(`section "${name}" not found`);
  }
  return section;
}

function getDateInputs() {
  const inputs = Array.from(document.querySelectorAll('input[type="date"]'));
  if (inputs.length < 2) {
    throw new Error('date inputs not found');
  }
  return inputs as HTMLInputElement[];
}

function renderPage(initialEntry = '/projects/1') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/projects/:id" element={<ProjectDetailPage />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('ProjectDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    setAuthUser();
    setProjectStore(true);

    mockedProjectService.getProjectMembers.mockResolvedValue({
      members: [
        {
          id: 1,
          user_id: 1,
          role_in_project: 'product',
          joined_at: NOW,
          is_owner: true,
          user: ownerUser,
        },
      ],
    });
    mockedProjectService.getProjectMemberCandidates.mockResolvedValue({
      users: [candidateUser],
    });
    mockedProjectService.addProjectMember.mockResolvedValue({});
    mockedProjectService.removeProjectMember.mockResolvedValue({});
    mockedProjectService.getSprints.mockResolvedValue({ sprints: [] });
    mockedProjectService.createSprint.mockResolvedValue({});
    mockedProjectService.updateSprintStatus.mockResolvedValue({});
    mockedProjectService.getBurndown.mockResolvedValue(burndownData);
    mockedProjectService.getVelocity.mockResolvedValue(velocityData);
    mockedProjectService.getQuality.mockResolvedValue(qualityData);
    mockedTechLeadService.getProjectTechLeads.mockResolvedValue({ tech_leads: [] });
    mockedTechLeadService.addTechLead.mockResolvedValue({});
    mockedTechLeadService.removeTechLead.mockResolvedValue({});
    mockedUserManagementService.getUsers.mockResolvedValue({ users: [], total: 0, page: 1, limit: 100 });
  });

  it('非法项目 ID 会直接展示错误，不触发数据加载', () => {
    renderPage('/projects/abc');

    expect(screen.getByText('项目ID无效')).toBeInTheDocument();
    expect(mockedProjectService.getProjectMembers).not.toHaveBeenCalled();
    expect(mockedProjectService.getSprints).not.toHaveBeenCalled();
  });

  it('项目 owner 会加载可添加成员候选并显示管理入口', async () => {
    renderPage();

    await waitFor(() => {
      expect(mockedProjectService.getProjectMemberCandidates).toHaveBeenCalledWith(1);
    });

    expect(screen.getByRole('button', { name: '添加成员' })).toBeInTheDocument();
    const userSelect = getSelectByDefaultOption('选择用户');
    expect(within(userSelect).getByRole('option', { name: /newdev@example.com/ })).toBeInTheDocument();
  });

  it('非 owner 不显示成员管理入口，也不会请求候选列表', async () => {
    setProjectStore(false);

    renderPage();

    await waitFor(() => {
      expect(mockedProjectService.getProjectMembers).toHaveBeenCalledWith(1);
    });

    expect(mockedProjectService.getProjectMemberCandidates).not.toHaveBeenCalled();
    expect(screen.queryByRole('button', { name: '添加成员' })).not.toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /newdev@example.com/ })).not.toBeInTheDocument();
  });

  it('添加成员后会刷新成员列表和候选列表', async () => {
    const user = userEvent.setup();

    mockedProjectService.getProjectMembers
      .mockResolvedValueOnce({
        members: [
          {
            id: 1,
            user_id: 1,
            role_in_project: 'product',
            joined_at: NOW,
            is_owner: true,
            user: ownerUser,
          },
        ],
      })
      .mockResolvedValueOnce({
        members: [
          {
            id: 1,
            user_id: 1,
            role_in_project: 'product',
            joined_at: NOW,
            is_owner: true,
            user: ownerUser,
          },
          {
            id: 2,
            user_id: 3,
            role_in_project: 'developer',
            joined_at: NOW,
            is_owner: false,
            user: candidateUser,
          },
        ],
      });
    mockedProjectService.getProjectMemberCandidates
      .mockResolvedValueOnce({ users: [candidateUser] })
      .mockResolvedValueOnce({ users: [] });

    renderPage();

    const userSelect = await waitFor(() => getSelectByDefaultOption('选择用户'));
    const roleSelect = getSelectByDefaultOption('产品经理');

    await user.selectOptions(userSelect, '3');
    await user.selectOptions(roleSelect, 'developer');
    await user.click(screen.getByRole('button', { name: '添加成员' }));

    await waitFor(() => {
      expect(mockedProjectService.addProjectMember).toHaveBeenCalledWith(1, {
        user_id: 3,
        role_in_project: 'developer',
      });
    });

    await waitFor(() => {
      expect(mockedProjectService.getProjectMembers).toHaveBeenCalledTimes(2);
      expect(mockedProjectService.getProjectMemberCandidates).toHaveBeenCalledTimes(2);
      expect(screen.getByText('newdev@example.com')).toBeInTheDocument();
    });

    expect(showSuccess).toHaveBeenCalledWith('项目成员添加成功');
  });

  it('移除普通成员时会要求确认，并在成功后刷新成员与候选列表', async () => {
    const user = userEvent.setup();
    mockedProjectService.getProjectMembers
      .mockResolvedValueOnce({
        members: [
          {
            id: 1,
            user_id: 1,
            role_in_project: 'product',
            joined_at: NOW,
            is_owner: true,
            user: ownerUser,
          },
          {
            id: 2,
            user_id: 3,
            role_in_project: 'developer',
            joined_at: NOW,
            is_owner: false,
            user: candidateUser,
          },
        ],
      })
      .mockResolvedValueOnce({
        members: [
          {
            id: 1,
            user_id: 1,
            role_in_project: 'product',
            joined_at: NOW,
            is_owner: true,
            user: ownerUser,
          },
        ],
      });
    mockedProjectService.getProjectMemberCandidates
      .mockResolvedValueOnce({ users: [] })
      .mockResolvedValueOnce({ users: [candidateUser] });

    renderPage();

    const memberSection = getSectionByHeading('项目成员');
    expect(await within(memberSection).findByText('newdev@example.com')).toBeInTheDocument();

    await user.click(within(memberSection).getByRole('button', { name: '移除' }));
    expect(screen.getByText('确认移除成员 newdev@example.com 吗？')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '确认移除' }));

    await waitFor(() => {
      expect(mockedProjectService.removeProjectMember).toHaveBeenCalledWith(1, 3);
    });

    await waitFor(() => {
      expect(mockedProjectService.getProjectMembers).toHaveBeenCalledTimes(2);
      expect(mockedProjectService.getProjectMemberCandidates).toHaveBeenCalledTimes(2);
      expect(screen.queryByText('newdev@example.com')).not.toBeInTheDocument();
    });

    expect(showSuccess).toHaveBeenCalledWith('成员移除成功');
  });

  it('admin 可以维护技术负责人，且候选列表会过滤为 tech lead 用户', async () => {
    const user = userEvent.setup();
    setAuthUser('admin');
    setProjectStore(false);
    mockedTechLeadService.getProjectTechLeads
      .mockResolvedValueOnce({ tech_leads: [] })
      .mockResolvedValueOnce({ tech_leads: [techLeadUser] })
      .mockResolvedValueOnce({ tech_leads: [] });
    mockedUserManagementService.getUsers.mockResolvedValue({
      users: [techLeadUser, anotherDeveloper],
      total: 2,
      page: 1,
      limit: 100,
    });

    renderPage();

    await waitFor(() => {
      expect(mockedUserManagementService.getUsers).toHaveBeenCalledWith({ page: 1, limit: 100 });
    });

    const techLeadSection = getSectionByHeading('项目技术负责人');
    const techLeadSelect = within(techLeadSection).getByRole('combobox');

    expect(
      within(techLeadSelect).getByRole('option', { name: 'techlead@example.com' })
    ).toBeInTheDocument();
    expect(
      within(techLeadSelect).queryByRole('option', { name: 'otherdev@example.com' })
    ).not.toBeInTheDocument();

    await user.click(within(techLeadSection).getByRole('button', { name: '添加技术负责人' }));
    expect(showError).toHaveBeenCalledWith('请选择技术负责人');

    await user.selectOptions(techLeadSelect, '4');
    await user.click(within(techLeadSection).getByRole('button', { name: '添加技术负责人' }));

    await waitFor(() => {
      expect(mockedTechLeadService.addTechLead).toHaveBeenCalledWith(1, 4);
    });

    const updatedTechLeadSection = getSectionByHeading('项目技术负责人');
    expect(await within(updatedTechLeadSection).findByText('techlead@example.com')).toBeInTheDocument();

    await user.click(within(updatedTechLeadSection).getByRole('button', { name: '移除' }));
    await user.click(screen.getByRole('button', { name: '确认移除' }));

    await waitFor(() => {
      expect(mockedTechLeadService.removeTechLead).toHaveBeenCalledWith(1, 4);
      expect(mockedTechLeadService.getProjectTechLeads).toHaveBeenCalledTimes(3);
    });

    expect(showSuccess).toHaveBeenCalledWith('技术负责人添加成功');
    expect(showSuccess).toHaveBeenCalledWith('技术负责人移除成功');
  });

  it('可以校验并创建冲刺，成功后刷新冲刺列表', async () => {
    const user = userEvent.setup();
    mockedProjectService.getSprints
      .mockResolvedValueOnce({ sprints: [] })
      .mockResolvedValueOnce({
        sprints: [
          {
            ...activeSprint,
            status: 'planned',
          },
        ],
      });

    renderPage();

    const sprintSection = getSectionByHeading('冲刺管理');
    await user.click(within(sprintSection).getByRole('button', { name: '+ 新建冲刺' }));
    await user.click(screen.getByRole('button', { name: '创建冲刺' }));
    expect(screen.getByText('请输入冲刺名称')).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText('例如：Sprint 1'), ' Sprint 1 ');
    const [startInput, endInput] = getDateInputs();
    fireEvent.change(startInput, { target: { value: '2026-04-10' } });
    fireEvent.change(endInput, { target: { value: '2026-04-01' } });

    await user.click(screen.getByRole('button', { name: '创建冲刺' }));
    expect(screen.getByText('结束日期不能早于开始日期')).toBeInTheDocument();

    fireEvent.change(endInput, { target: { value: '2026-04-20' } });
    await user.click(screen.getByRole('button', { name: '创建冲刺' }));

    await waitFor(() => {
      expect(mockedProjectService.createSprint).toHaveBeenCalledWith(1, {
        name: 'Sprint 1',
        goal: '',
        start_date: '2026-04-10',
        end_date: '2026-04-20',
      });
    });

    await waitFor(() => {
      expect(mockedProjectService.getSprints).toHaveBeenCalledTimes(2);
      expect(within(getSectionByHeading('冲刺管理')).getByText('Sprint 1')).toBeInTheDocument();
    });

    expect(showSuccess).toHaveBeenCalledWith('冲刺创建成功');
    expect(screen.queryByText('新建冲刺')).not.toBeInTheDocument();
  });

  it('会渲染燃尽图和报表数据，并支持刷新报表及完成冲刺', async () => {
    const user = userEvent.setup();
    mockedProjectService.getSprints
      .mockResolvedValueOnce({ sprints: [activeSprint] })
      .mockResolvedValueOnce({ sprints: [completedSprint] });
    mockedProjectService.getVelocity.mockResolvedValue(richVelocityData);
    mockedProjectService.getQuality.mockResolvedValue(richQualityData);

    renderPage();

    const velocitySection = getSectionByHeading('速度报表');
    const qualitySection = getSectionByHeading('质量报表');
    const burndownSection = getSectionByHeading('燃尽图');

    expect(await within(burndownSection).findByText('当前剩余（实际）')).toBeInTheDocument();
    expect(within(velocitySection).getByText(/速度 75\.0%/)).toBeInTheDocument();
    expect(within(qualitySection).getByText('缺陷状态分布')).toBeInTheDocument();
    expect(within(qualitySection).getByText('严重')).toBeInTheDocument();
    expect(within(burndownSection).getByText('已燃尽: 12 点')).toBeInTheDocument();

    await user.click(within(velocitySection).getByRole('button', { name: '刷新' }));

    await waitFor(() => {
      expect(mockedProjectService.getVelocity).toHaveBeenCalledTimes(2);
      expect(mockedProjectService.getQuality).toHaveBeenCalledTimes(2);
    });

    const sprintSection = getSectionByHeading('冲刺管理');
    await user.click(within(sprintSection).getByRole('button', { name: '完成冲刺' }));

    await waitFor(() => {
      expect(mockedProjectService.updateSprintStatus).toHaveBeenCalledWith(31, {
        status: 'completed',
      });
      expect(mockedProjectService.getBurndown.mock.calls.length).toBeGreaterThanOrEqual(2);
    });

    expect(showSuccess).toHaveBeenCalledWith('冲刺已更新为已完成');
  });
});
