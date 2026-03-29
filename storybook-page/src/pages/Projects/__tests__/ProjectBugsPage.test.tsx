import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import { bugService } from '../../../services/bugService';
import { projectService } from '../../../services/projectService';
import { storyService } from '../../../services/storyService';
import ProjectBugsPage from '../ProjectBugsPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/bugService', () => ({
  bugService: {
    getProjectBugs: vi.fn(),
    getBug: vi.fn(),
    createBug: vi.fn(),
    updateBugStatus: vi.fn(),
    assignBug: vi.fn(),
  },
}));

vi.mock('../../../services/projectService', () => ({
  projectService: {
    getProjectMembers: vi.fn(),
  },
}));

vi.mock('../../../services/storyService', () => ({
  storyService: {
    getStories: vi.fn(),
  },
}));

const mockedBugService = vi.mocked(bugService, { deep: true });
const mockedProjectService = vi.mocked(projectService, { deep: true });
const mockedStoryService = vi.mocked(storyService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

const members = [
  {
    id: 1,
    user_id: 11,
    role_in_project: 'developer',
    user: {
      id: 11,
      email: 'dev@example.com',
      role: 'developer',
      created_at: NOW,
    } satisfies User,
  },
  {
    id: 2,
    user_id: 12,
    role_in_project: 'tester',
    user: {
      id: 12,
      email: 'tester@example.com',
      role: 'tester',
      created_at: NOW,
    } satisfies User,
  },
  {
    id: 3,
    user_id: 13,
    role_in_project: 'product',
    user: {
      id: 13,
      email: 'product@example.com',
      role: 'product',
      created_at: NOW,
    } satisfies User,
  },
  {
    id: 4,
    user_id: 14,
    role_in_project: 'admin',
    user: {
      id: 14,
      email: 'admin@example.com',
      role: 'admin',
      created_at: NOW,
    } satisfies User,
  },
];

function setAuthUser(role: User['role']) {
  useAuthStore.setState({
    user: {
      id: 99,
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

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/projects/1/bugs']}>
      <Routes>
        <Route path="/projects/:id/bugs" element={<ProjectBugsPage />} />
      </Routes>
    </MemoryRouter>
  );
}

function getFieldSelect(label: string, occurrence = 0) {
  const labelNode = screen.getAllByText(label)[occurrence];
  const select = labelNode.parentElement?.querySelector('select');
  if (!(select instanceof HTMLSelectElement)) {
    throw new Error(`select for "${label}" not found`);
  }
  return select;
}

describe('ProjectBugsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();

    mockedProjectService.getProjectMembers.mockResolvedValue({ members });
    mockedStoryService.getStories.mockResolvedValue({
      stories: [
        {
          id: 7,
          title: '登录故事',
        },
      ],
    });
    mockedBugService.getProjectBugs.mockResolvedValue({
      bugs: [
        {
          id: 101,
          project_id: 1,
          story_id: 7,
          title: '登录缺陷',
          severity: 'high',
          status: 'open',
          assigned_to: 11,
          created_at: NOW,
          updated_at: NOW,
        },
      ],
    });
  });

  it('tester 能创建缺陷，且初始负责人只包含 developer/admin', async () => {
    setAuthUser('tester');
    renderPage();

    await waitFor(() => {
      expect(mockedProjectService.getProjectMembers).toHaveBeenCalledWith(1);
    });

    expect(screen.getByRole('button', { name: '创建缺陷' })).toBeInTheDocument();
    const assigneeSelect = getFieldSelect('初始负责人');

    expect(within(assigneeSelect).getByRole('option', { name: /dev@example.com/ })).toBeInTheDocument();
    expect(within(assigneeSelect).getByRole('option', { name: /admin@example.com/ })).toBeInTheDocument();
    expect(
      within(assigneeSelect).queryByRole('option', { name: /tester@example.com/ })
    ).not.toBeInTheDocument();
    expect(
      within(assigneeSelect).queryByRole('option', { name: /product@example.com/ })
    ).not.toBeInTheDocument();
    expect(screen.getAllByRole('option', { name: '处理中' })).toHaveLength(2);
  });

  it('product 不显示创建表单，但会看到可编辑的指派控件', async () => {
    setAuthUser('product');
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('登录缺陷')).toBeInTheDocument();
    });

    expect(screen.getByText('当前角色没有新建缺陷权限。')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '创建缺陷' })).not.toBeInTheDocument();
    expect(screen.getAllByRole('option', { name: '处理中' })).toHaveLength(1);
    expect(screen.getAllByRole('option', { name: /dev@example.com/ })).toHaveLength(2);
  });

  it('developer 只能编辑状态，指派控件保持只读', async () => {
    setAuthUser('developer');
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('登录缺陷')).toBeInTheDocument();
    });

    expect(screen.getByText('当前角色没有新建缺陷权限。')).toBeInTheDocument();
    expect(screen.getAllByRole('option', { name: '处理中' })).toHaveLength(2);
    expect(screen.getAllByRole('option', { name: /dev@example.com/ })).toHaveLength(1);
    expect(screen.getAllByText('dev@example.com')[0]).toBeInTheDocument();
  });

  it('tester 创建缺陷成功后会刷新列表并清空表单', async () => {
    const user = userEvent.setup();
    setAuthUser('tester');
    mockedBugService.createBug.mockResolvedValue({});

    renderPage();

    await screen.findByRole('button', { name: '创建缺陷' });

    await user.type(screen.getByPlaceholderText('一句话说清问题'), ' 登录失败 ');
    await user.type(screen.getByPlaceholderText('补充复现方式、影响范围或截图说明'), ' 点击无响应 ');
    await user.selectOptions(getFieldSelect('严重级别', 0), 'critical');
    await user.selectOptions(getFieldSelect('关联故事'), '7');
    await user.selectOptions(getFieldSelect('初始负责人'), '11');
    await user.click(screen.getByRole('button', { name: '创建缺陷' }));

    await waitFor(() => {
      expect(mockedBugService.createBug).toHaveBeenCalledWith(1, {
        title: '登录失败',
        description: '点击无响应',
        severity: 'critical',
        story_id: 7,
        assigned_to: 11,
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('缺陷创建成功');
    await waitFor(() => {
      expect(mockedBugService.getProjectBugs).toHaveBeenCalledTimes(2);
    });
    expect(screen.getByPlaceholderText('一句话说清问题')).toHaveValue('');
  });

  it('product 可以按条件筛选列表、清空筛选并查看缺陷详情', async () => {
    const user = userEvent.setup();
    setAuthUser('product');
    mockedBugService.getBug.mockResolvedValue({
      id: 101,
      project_id: 1,
      story_id: 7,
      title: '登录缺陷',
      description: '详情描述',
      severity: 'high',
      status: 'open',
      assigned_to: {
        id: 11,
        email: 'dev@example.com',
      },
      created_at: NOW,
      updated_at: NOW,
    });

    renderPage();

    await screen.findByText('登录缺陷');

    await user.selectOptions(getFieldSelect('状态'), 'open');
    await user.selectOptions(getFieldSelect('严重级别'), 'high');
    await user.selectOptions(getFieldSelect('负责人'), '11');

    await waitFor(() => {
      expect(mockedBugService.getProjectBugs).toHaveBeenLastCalledWith(1, {
        status: 'open',
        severity: 'high',
        assignee: 11,
      });
    });

    await user.click(screen.getByRole('button', { name: '清空' }));

    await waitFor(() => {
      expect(mockedBugService.getProjectBugs).toHaveBeenLastCalledWith(1, {
        status: undefined,
        severity: undefined,
        assignee: undefined,
      });
    });

    await user.click(screen.getByRole('button', { name: '查看详情' }));

    await waitFor(() => {
      expect(mockedBugService.getBug).toHaveBeenCalledWith(101);
    });

    expect(await screen.findByText('详情描述')).toBeInTheDocument();
  });

  it('admin 可以更新缺陷状态并重新指派负责人', async () => {
    const user = userEvent.setup();
    setAuthUser('admin');
    mockedBugService.updateBugStatus.mockResolvedValue({});
    mockedBugService.assignBug.mockResolvedValue({
      assigned_to: {
        id: 14,
        email: 'admin@example.com',
      },
    });

    renderPage();

    await screen.findByText('登录缺陷');

    const rowSelects = screen
      .getAllByRole('combobox')
      .filter((element) => element.className.includes('field-control'));
    const statusSelect = rowSelects.at(-2);
    const assigneeSelect = rowSelects.at(-1);
    if (!(statusSelect instanceof HTMLSelectElement) || !(assigneeSelect instanceof HTMLSelectElement)) {
      throw new Error('row selects not found');
    }

    await user.selectOptions(statusSelect, 'resolved');
    await waitFor(() => {
      expect(mockedBugService.updateBugStatus).toHaveBeenCalledWith(101, {
        status: 'resolved',
      });
    });

    await user.selectOptions(assigneeSelect, '14');
    await waitFor(() => {
      expect(mockedBugService.assignBug).toHaveBeenCalledWith(101, {
        assigned_to: 14,
      });
    });
    expect(showSuccess).toHaveBeenCalledWith('缺陷指派已更新');
  });

  it('基础数据或详情加载失败时会提示错误', async () => {
    const user = userEvent.setup();
    setAuthUser('tester');
    mockedProjectService.getProjectMembers.mockRejectedValueOnce(new Error('成员接口失败'));
    mockedBugService.getBug.mockRejectedValueOnce(new Error('详情接口失败'));

    renderPage();

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('成员接口失败');
    });

    await screen.findByText('登录缺陷');
    await user.click(screen.getByRole('button', { name: '查看详情' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('详情接口失败');
    });
    await waitFor(() => {
      expect(screen.queryByText('缺陷详情')).not.toBeInTheDocument();
    });
  });
});
