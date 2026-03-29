import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Story, User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import { useStoryStore } from '../../../stores/storyStore';
import { projectService } from '../../../services/projectService';
import { storyService } from '../../../services/storyService';
import { aiService } from '../../../services/aiService';
import StoryDetailPage from '../StoryDetailPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../components/story/AcceptanceCriteriaList', () => ({
  default: ({ criteria }: { criteria: Array<{ description: string }> }) => (
    <div>AC:{criteria.length}</div>
  ),
}));

vi.mock('../../../components/story/ActivityTimeline', () => ({
  default: ({ activities }: { activities: Array<unknown> }) => <div>Activity:{activities.length}</div>,
}));

vi.mock('../../../components/story/StoryTasksPanel', () => ({
  default: ({ storyId }: { storyId: number }) => <div>Tasks:{storyId}</div>,
}));

vi.mock('../../../components/story/StoryTestCasesPanel', () => ({
  default: ({ storyId }: { storyId: number }) => <div>Cases:{storyId}</div>,
}));

vi.mock('../../../components/story/StoryForm', () => ({
  default: ({ isOpen }: { isOpen: boolean }) => (isOpen ? <div>StoryForm Open</div> : null),
}));

vi.mock('../../../services/projectService', () => ({
  projectService: {
    getSprints: vi.fn(),
    getProjectMembers: vi.fn(),
  },
}));

vi.mock('../../../services/storyService', () => ({
  storyService: {
    planToSprint: vi.fn(),
    assignStory: vi.fn(),
  },
}));

vi.mock('../../../services/aiService', () => ({
  aiService: {
    checkInvest: vi.fn(),
    splitStory: vi.fn(),
  },
}));

const mockedProjectService = vi.mocked(projectService, { deep: true });
const mockedStoryService = vi.mocked(storyService, { deep: true });
const mockedAIService = vi.mocked(aiService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function createStory(overrides: Partial<Story> = {}): Story {
  return {
    id: 18,
    project_id: 4,
    project: { id: 4, name: 'Alpha' },
    title: '登录故事',
    description: '实现登录',
    story_type: 'feature',
    status: 'ready',
    review_status: 'approved',
    priority: 2,
    story_points: 3,
    position: 1024,
    assigned_to: undefined,
    created_by: {
      id: 7,
      email: 'creator@example.com',
      role: 'product',
      created_at: NOW,
    },
    acceptance_criteria: [
      {
        id: 'ac-1',
        description: 'Given 用户已注册',
        status: 'pending',
        order: 1,
      },
    ],
    tags: ['auth'],
    created_at: NOW,
    updated_at: NOW,
    ...overrides,
  };
}

function setAuthUser(role: User['role'], id = 99) {
  useAuthStore.setState({
    user: {
      id,
      email: `${role}@example.com`,
      role,
      created_at: NOW,
    },
    token: 'token',
    isAuthenticated: true,
    isLoading: false,
    error: null,
  });
}

function setStoryStore(story: Story) {
  useStoryStore.setState({
    boardData: {
      pending: [],
      backlog: [],
      ready: [],
      in_progress: [],
      test: [],
      done: [],
    },
    currentStory: story,
    activities: [],
    isLoading: false,
    isUpdating: false,
    error: null,
    fetchBoardData: vi.fn(async () => {}),
    fetchStory: vi.fn(async () => {}),
    fetchActivities: vi.fn(async () => {}),
    updateStoryStatus: vi.fn(async () => {}),
    claimStory: vi.fn(async () => {}),
    releaseStory: vi.fn(async () => {}),
    updateACStatus: vi.fn(async () => {}),
    clearCurrentStory: vi.fn(),
    clearError: vi.fn(),
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/stories/18']}>
      <Routes>
        <Route path="/stories/:id" element={<StoryDetailPage />} />
      </Routes>
    </MemoryRouter>
  );
}

function getAssigneeSelect() {
  const select = screen
    .getAllByRole('combobox')
    .find((element) => element.querySelector('option[value="11"]'));

  if (!(select instanceof HTMLSelectElement)) {
    throw new Error('assignee select not found');
  }

  return select;
}

function getSprintSelect() {
  const select = screen
    .getAllByRole('combobox')
    .find((element) => element.querySelector('option')?.textContent?.includes('不加入冲刺'));

  if (!(select instanceof HTMLSelectElement)) {
    throw new Error('sprint select not found');
  }

  return select;
}

describe('StoryDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();

    mockedProjectService.getSprints.mockResolvedValue({ sprints: [] });
    mockedProjectService.getProjectMembers.mockResolvedValue({
      members: [
        {
          id: 1,
          user_id: 11,
          role_in_project: 'developer',
          user: { id: 11, email: 'dev@example.com' },
        },
      ],
    });
    mockedStoryService.assignStory.mockResolvedValue({});
    mockedStoryService.planToSprint.mockResolvedValue({
      story_id: 18,
      sprint_id: 3,
      updated_at: NOW,
    });
    mockedAIService.checkInvest.mockResolvedValue({
      story_id: 18,
      invest_score: 0.85,
      checks: {
        independent: { score: 0.8, status: 'pass', title: 'Independent', description: '' },
        negotiable: { score: 0.8, status: 'pass', title: 'Negotiable', description: '' },
        valuable: { score: 0.9, status: 'pass', title: 'Valuable', description: '' },
        estimable: { score: 0.8, status: 'pass', title: 'Estimable', description: '' },
        small: { score: 0.9, status: 'pass', title: 'Small', description: '' },
        testable: { score: 0.9, status: 'pass', title: 'Testable', description: '' },
      },
      suggestions: [],
    });
    mockedAIService.splitStory.mockResolvedValue({
      story_id: 18,
      origin_title: '登录故事',
      sub_stories: [],
    });
  });

  it('admin 在未分配故事上既能分配负责人，也能直接领取故事', async () => {
    const user = userEvent.setup();
    const claimStory = vi.fn(async () => {});

    setAuthUser('admin');
    setStoryStore(createStory());
    useStoryStore.setState({ claimStory });

    renderPage();

    await waitFor(() => {
      expect(mockedProjectService.getProjectMembers).toHaveBeenCalledWith(4);
    });

    expect(screen.getByRole('button', { name: '领取故事' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存分配' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '领取故事' }));

    await waitFor(() => {
      expect(claimStory).toHaveBeenCalledWith(18);
      expect(showSuccess).toHaveBeenCalledWith('故事领取成功');
    });
  });

  it('admin 对他人已领取的故事也能看到释放入口，并调用 release 接口', async () => {
    const user = userEvent.setup();
    const releaseStory = vi.fn(async () => {});

    setAuthUser('admin');
    setStoryStore(
      createStory({
        status: 'in_progress',
        assigned_to: {
          id: 11,
          email: 'dev@example.com',
          role: 'developer',
          created_at: NOW,
        },
      })
    );
    useStoryStore.setState({ releaseStory });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '释放' })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: '释放' }));

    await waitFor(() => {
      expect(releaseStory).toHaveBeenCalledWith(18);
      expect(showSuccess).toHaveBeenCalledWith('故事已释放');
      expect(mockedStoryService.assignStory).not.toHaveBeenCalled();
    });
  });

  it('product 可以加载开发成员并保存负责人分配', async () => {
    const user = userEvent.setup();
    const fetchStory = vi.fn(async () => {});

    setAuthUser('product', 7);
    setStoryStore(createStory());
    useStoryStore.setState({ fetchStory });

    renderPage();

    const assigneeSelect = await waitFor(() => getAssigneeSelect());
    await user.selectOptions(assigneeSelect, '11');
    await user.click(screen.getByRole('button', { name: '保存分配' }));

    await waitFor(() => {
      expect(mockedStoryService.assignStory).toHaveBeenCalledWith(18, {
        assigned_to: 11,
      });
      expect(fetchStory).toHaveBeenCalledWith(18);
      expect(showSuccess).toHaveBeenCalledWith('负责人分配成功');
    });
  });

  it('product 可以规划冲刺并触发规则辅助结果展示', async () => {
    const user = userEvent.setup();
    const fetchStory = vi.fn(async () => {});

    mockedProjectService.getSprints.mockResolvedValueOnce({
      sprints: [
        {
          id: 3,
          name: 'Sprint 1',
          status: 'planned',
          start_date: NOW,
          end_date: NOW,
        },
      ],
    });
    mockedAIService.splitStory.mockResolvedValueOnce({
      story_id: 18,
      origin_title: '登录故事',
      sub_stories: [
        {
          title: '拆分后的子故事',
          description: '拆分建议',
          story_points: 2,
          acceptance_criteria: [{ id: 'ac-2', description: '可独立交付', status: 'pending' }],
        },
      ],
    });

    setAuthUser('product', 7);
    setStoryStore(createStory());
    useStoryStore.setState({ fetchStory });

    renderPage();

    const sprintSelect = await waitFor(() => getSprintSelect());
    await user.selectOptions(sprintSelect, '3');

    await waitFor(() => {
      expect(mockedStoryService.planToSprint).toHaveBeenCalledWith(18, { sprint_id: 3 });
      expect(fetchStory).toHaveBeenCalledWith(18);
    });

    await user.click(screen.getByRole('button', { name: '执行检查' }));

    await waitFor(() => {
      expect(mockedAIService.checkInvest).toHaveBeenCalledWith(18);
    });
    expect(await screen.findByText('Independent')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '生成拆分建议' }));

    await waitFor(() => {
      expect(mockedAIService.splitStory).toHaveBeenCalledWith(18, { target_count: 3 });
    });

    expect(await screen.findByText('拆分后的子故事')).toBeInTheDocument();
  });

  it('管理员会看到冲刺/成员加载错误，并透传规则辅助失败信息', async () => {
    const user = userEvent.setup();

    mockedProjectService.getSprints.mockRejectedValueOnce(new Error('load sprints failed'));
    mockedProjectService.getProjectMembers.mockRejectedValueOnce(new Error('load members failed'));
    mockedAIService.checkInvest.mockRejectedValueOnce(new Error('INVEST 服务异常'));
    mockedAIService.splitStory.mockRejectedValueOnce(new Error('拆分服务异常'));

    setAuthUser('admin');
    setStoryStore(createStory());

    renderPage();

    expect(await screen.findByText('冲刺列表加载失败')).toBeInTheDocument();
    expect(await screen.findByText('成员列表加载失败')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '执行检查' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('INVEST 服务异常');
    });
    expect(screen.getByText('INVEST 服务异常')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '生成拆分建议' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('拆分服务异常');
    });
    expect(screen.getByText('拆分服务异常')).toBeInTheDocument();
  });
});
