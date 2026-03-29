import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Project, User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import { useProjectStore } from '../../../stores/projectStore';
import BoardViewPage from '../BoardViewPage';

vi.mock('../../../components/board/KanbanBoard', () => ({
  default: ({ projectId }: { projectId: number }) => <div>KanbanBoard:{projectId}</div>,
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
    localStorage.clear();
    setAuthUser('product');
    setProjectStore();
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
});
