import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Project, User } from '../../../types/models';
import { useProjectStore } from '../../../stores/projectStore';
import ProjectListPage from '../ProjectListPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

const NOW = '2026-03-29T00:00:00Z';

const owner: User = {
  id: 1,
  email: 'owner@example.com',
  role: 'product',
  created_at: NOW,
};

function createProject(id: number, overrides: Partial<Project> = {}): Project {
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
    createProject: vi.fn(async () => createProject(99)),
    updateProject: vi.fn(async () => {}),
    deleteProject: vi.fn(async () => {}),
    setCurrentProject: vi.fn(),
    clearError: vi.fn(),
    ...overrides,
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/projects']}>
      <Routes>
        <Route path="/projects" element={<ProjectListPage />} />
        <Route path="/projects/:id" element={<div>Project Detail Page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('ProjectListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setProjectStore();
  });

  it('挂载时会拉取项目，并支持按名称或描述搜索', async () => {
    const user = userEvent.setup();
    const fetchProjects = vi.fn(async () => {});

    setProjectStore({
      projects: [
        createProject(1, { name: 'Alpha', description: '认证与权限' }),
        createProject(2, { name: 'Beta', description: '支付链路重构', agile_mode: 'scrum' }),
      ],
      fetchProjects,
    });

    renderPage();

    expect(fetchProjects).toHaveBeenCalledTimes(1);
    expect(screen.getByText('Alpha')).toBeInTheDocument();
    expect(screen.getByText('Beta')).toBeInTheDocument();

    await user.type(screen.getByLabelText('搜索'), '支付');

    expect(screen.getByText('当前筛出 1 个结果')).toBeInTheDocument();
    expect(screen.queryByText('Alpha')).not.toBeInTheDocument();
    expect(screen.getByText('Beta')).toBeInTheDocument();
  });

  it('创建项目时会校验项目名称长度', async () => {
    const user = userEvent.setup();
    const createProjectAction = vi.fn(async () => createProject(9));

    setProjectStore({
      projects: [createProject(1)],
      createProject: createProjectAction,
    });

    renderPage();

    await user.click(screen.getByRole('button', { name: '+ 新建项目' }));
    await user.click(screen.getByRole('button', { name: '创建项目' }));

    expect(await screen.findByText('项目名称长度应在2-100字符之间')).toBeInTheDocument();
    expect(createProjectAction).not.toHaveBeenCalled();
  });

  it('创建项目成功后会提示并跳转到项目详情页', async () => {
    const user = userEvent.setup();
    const createProjectAction = vi.fn(async () =>
      createProject(9, {
        name: 'Gamma',
        description: '新的交付空间',
      })
    );

    setProjectStore({
      projects: [createProject(1)],
      createProject: createProjectAction,
    });

    renderPage();

    await user.click(screen.getByRole('button', { name: '+ 新建项目' }));
    await user.type(screen.getByLabelText(/项目名称/), 'Gamma');
    await user.type(screen.getByLabelText('项目描述'), '新的交付空间');
    await user.click(screen.getByRole('button', { name: '创建项目' }));

    await waitFor(() => {
      expect(createProjectAction).toHaveBeenCalledWith({
        name: 'Gamma',
        description: '新的交付空间',
        agile_mode: 'kanban',
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('项目创建成功');
    expect(await screen.findByText('Project Detail Page')).toBeInTheDocument();
  });

  it('创建项目失败时会透传错误提示', async () => {
    const user = userEvent.setup();
    const createProjectAction = vi.fn(async () => {
      throw new Error('创建服务异常');
    });

    setProjectStore({
      projects: [createProject(1)],
      createProject: createProjectAction,
    });

    renderPage();

    await user.click(screen.getByRole('button', { name: '+ 新建项目' }));
    await user.type(screen.getByLabelText(/项目名称/), 'Gamma');
    await user.click(screen.getByRole('button', { name: '创建项目' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('创建服务异常');
    });
  });
});
