import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { DashboardData } from '../../../types/api';
import type { Project, User } from '../../../types/models';
import apiClient from '../../../services/api';
import { useAuthStore } from '../../../stores/authStore';
import { useProjectStore } from '../../../stores/projectStore';
import DashboardPage from '../DashboardPage';

vi.mock('../../../services/api', () => ({
  default: {
    get: vi.fn(),
  },
}));

const mockedApiClient = vi.mocked(apiClient, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function createProject(id: number, overrides: Partial<Project> = {}): Project {
  const owner: User = {
    id: 1,
    email: 'owner@example.com',
    role: 'product',
    created_at: NOW,
  };

  return {
    id,
    name: `Project ${id}`,
    description: `Description ${id}`,
    agile_mode: 'kanban',
    owner,
    member_count: 3,
    story_count: 5,
    is_owner: id === 1,
    created_at: NOW,
    updated_at: NOW,
    ...overrides,
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

function buildDashboardData(): DashboardData {
  return {
    user: {
      id: 7,
      email: 'product@example.com',
      role: 'product',
      created_at: NOW,
    },
    my_stories: {
      assigned: [
        {
          id: 18,
          title: '登录故事',
          project: 'Alpha',
          status: 'in_progress',
          priority: 1,
          story_type: 'feature',
        },
      ],
      created: [
        {
          id: 19,
          title: '支付故事',
          project: 'Beta',
          status: 'pending',
          priority: 2,
          story_type: 'feature',
        },
      ],
    },
    statistics: {
      total_assigned: 3,
      in_progress: 1,
      completed: 2,
    },
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <Routes>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/projects" element={<div>Projects Page</div>} />
        <Route path="/projects/:id" element={<div>Project Detail Page</div>} />
        <Route path="/projects/:id/board" element={<div>Board Page</div>} />
        <Route path="/projects/:projectId/stories/new" element={<div>Story Create Page</div>} />
        <Route path="/stories/:id" element={<div>Story Detail Page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    setAuthUser('product');
    setProjectStore();
  });

  it('加载工作台数据后会展示快速开始入口，并支持切换目标项目', async () => {
    const user = userEvent.setup();
    const fetchProjects = vi.fn(async () => {});

    setProjectStore({
      projects: [
        createProject(1, { name: 'Alpha' }),
        createProject(2, { name: 'Beta', agile_mode: 'scrum', is_owner: false }),
      ],
      fetchProjects,
    });
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: buildDashboardData(),
      },
    });

    renderPage();

    await waitFor(() => {
      expect(mockedApiClient.get).toHaveBeenCalledWith('/api/me/dashboard');
      expect(fetchProjects).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByText('我领取的故事')).toBeInTheDocument();
    expect(screen.getByText('登录故事')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /新建故事/ })).toHaveAttribute(
      'href',
      '/projects/1/stories/new'
    );

    await user.selectOptions(screen.getByRole('combobox'), '2');

    expect(screen.getByText('以下操作将基于「Beta」')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /新建故事/ })).toHaveAttribute(
      'href',
      '/projects/2/stories/new'
    );
  });

  it('接口失败时会展示加载错误', async () => {
    mockedApiClient.get.mockRejectedValue(new Error('network down'));

    renderPage();

    expect(await screen.findByText('工作台数据加载失败，请稍后重试')).toBeInTheDocument();
  });

  it('没有项目时会引导去创建项目，且 developer 不显示新建故事入口', async () => {
    setAuthUser('developer');
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: buildDashboardData(),
      },
    });

    renderPage();

    expect(await screen.findByText('快速开始')).toBeInTheDocument();
    expect(screen.getByText('浏览与协作')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '去创建项目' })).toHaveAttribute(
      'href',
      '/projects'
    );
    expect(screen.queryByRole('link', { name: /新建故事/ })).not.toBeInTheDocument();
  });
});
