import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Project, StoryBoardItem, User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import { useProjectStore } from '../../../stores/projectStore';
import { useStoryStore } from '../../../stores/storyStore';
import { projectService } from '../../../services/projectService';
import BoardViewPage from '../BoardViewPage';

const kanbanPropsStack: Array<Record<string, unknown>> = [];

vi.mock('../../../components/board/KanbanBoard', () => ({
  default: (props: { projectId: number }) => {
    kanbanPropsStack.push(props);
    return <div>KanbanBoard:{props.projectId}</div>;
  },
}));

vi.mock('../../../services/projectService', () => ({
  projectService: {
    getSprints: vi.fn(),
  },
}));

vi.mock('../../../components/story/StoryForm', () => ({
  default: ({
    isOpen,
    projectId,
    onClose,
  }: {
    isOpen: boolean;
    projectId: number;
    onClose: () => void;
  }) =>
    isOpen ? (
      <div>
        <span>StoryForm:{projectId}</span>
        <button onClick={onClose}>关闭故事表单</button>
      </div>
    ) : null,
}));

const NOW = '2026-03-29T00:00:00Z';

const owner: User = {
  id: 1,
  email: 'owner@example.com',
  role: 'product',
  created_at: NOW,
};

function createProject(id: number): Project {
  return {
    id,
    name: 'Alpha',
    description: '核心项目',
    agile_mode: 'kanban',
    owner,
    is_owner: true,
    created_at: NOW,
    updated_at: NOW,
  };
}

function setAuthUser(role: User['role']) {
  useAuthStore.setState({
    user: {
      id: 7,
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

function setProjectStore(overrides: Partial<ReturnType<typeof useProjectStore.getState>> = {}) {
  useProjectStore.setState({
    currentProject: null,
    projectOverview: null,
    projects: [],
    isLoading: false,
    error: null,
    fetchProjects: vi.fn(async () => {}),
    fetchProject: vi.fn(async () => {}),
    fetchProjectOverview: vi.fn(async () => {}),
    createProject: vi.fn(async () => createProject(9)),
    updateProject: vi.fn(async () => {}),
    deleteProject: vi.fn(async () => {}),
    setCurrentProject: vi.fn(),
    clearError: vi.fn(),
    ...overrides,
  });
}

function renderPage(initialEntry = '/projects/1/board') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/projects/:id/board" element={<BoardViewPage />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('BoardViewPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    kanbanPropsStack.length = 0;
    localStorage.clear();
    setAuthUser('product');
    setProjectStore();
    useStoryStore.setState({
      boardData: {
        pending: [],
        backlog: [],
        ready: [],
        in_progress: [],
        test: [],
        done: [],
      },
      currentStory: null,
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
    vi.mocked(projectService.getSprints).mockResolvedValue({ sprints: [] });
  });

  it('非法项目 ID 会直接提示错误而不是一直停留在加载中', () => {
    const fetchProject = vi.fn(async () => {});

    setProjectStore({ fetchProject });

    renderPage('/projects/abc/board');

    expect(screen.getByText('项目ID无效')).toBeInTheDocument();
    expect(fetchProject).not.toHaveBeenCalled();
  });

  it('product 可以打开和关闭创建故事表单', async () => {
    const user = userEvent.setup();
    const fetchProject = vi.fn(async () => {});

    setProjectStore({
      currentProject: createProject(1),
      fetchProject,
    });

    renderPage();

    await waitFor(() => {
      expect(fetchProject).toHaveBeenCalledWith(1);
    });

    expect(screen.getByText('KanbanBoard:1')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '创建故事' }));
    expect(screen.getByText('StoryForm:1')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '关闭故事表单' }));
    expect(screen.queryByText('StoryForm:1')).not.toBeInTheDocument();
  });

  it('非产品角色只显示权限提示，不显示创建入口', () => {
    setAuthUser('developer');
    setProjectStore({
      currentProject: createProject(1),
    });

    renderPage();

    expect(screen.getByText('仅产品经理和管理员可创建故事')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '创建故事' })).not.toBeInTheDocument();
  });

  it('筛选栏渲染负责人选项，输入关键词会透传给看板', async () => {
    const user = userEvent.setup();
    setProjectStore({
      currentProject: createProject(1),
    });

    const backlogStory: StoryBoardItem = {
      id: 31,
      title: '支持手机号登录',
      story_type: 'feature',
      status: 'backlog',
      priority: 2,
      position: 1,
      assigned_to: { id: 9, email: 'dev@example.com', role: 'developer', created_at: NOW },
    };
    useStoryStore.setState({
      boardData: {
        pending: [],
        backlog: [backlogStory],
        ready: [],
        in_progress: [],
        test: [],
        done: [],
      },
    });

    renderPage();

    const assigneeSelect = (await screen.findByLabelText('负责人')) as HTMLSelectElement;
    await waitFor(() => {
      expect(assigneeSelect).not.toBeNull();
    });

    // 负责人选项来自看板数据聚合
    expect(
      Array.from(assigneeSelect.options).some((option) => option.textContent === 'dev@example.com')
    ).toBe(true);

    await user.type(screen.getByLabelText('关键词'), '手机号');

    const latestProps = kanbanPropsStack[kanbanPropsStack.length - 1] as {
      filters?: { keyword?: string };
    };
    expect(latestProps.filters?.keyword).toBe('手机号');
  });
});
