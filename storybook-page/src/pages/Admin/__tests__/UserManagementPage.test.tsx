import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { User } from '../../../types/models';
import { userManagementService } from '../../../services/userManagementService';
import UserManagementPage from '../UserManagementPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/userManagementService', () => ({
  userManagementService: {
    getUsers: vi.fn(),
    createUser: vi.fn(),
    updateUser: vi.fn(),
    deleteUser: vi.fn(),
    getUserWorkload: vi.fn(),
  },
}));

const mockedUserManagementService = vi.mocked(userManagementService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function createUser(id: number, overrides: Partial<User> = {}): User {
  return {
    id,
    email: `user${id}@example.com`,
    role: 'developer',
    created_at: NOW,
    ...overrides,
  };
}

describe('UserManagementPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockedUserManagementService.getUsers.mockResolvedValue({
      users: [createUser(1), createUser(2, { email: 'tester@example.com', role: 'tester' })],
      total: 2,
      page: 1,
      limit: 20,
    });
    mockedUserManagementService.createUser.mockResolvedValue({});
    mockedUserManagementService.deleteUser.mockResolvedValue({});
    mockedUserManagementService.updateUser.mockResolvedValue({});
    mockedUserManagementService.getUserWorkload.mockResolvedValue({
      user: createUser(1),
      active_stories: [{ id: 18, title: '登录故事', status: 'in_progress' }],
      active_tasks: [{ id: 28, title: '实现接口', status: 'todo' }],
      statistics: {
        total_story_points: 8,
        total_estimated_hours: 12,
        stories_completed_30d: 1,
        stories_in_progress: 1,
        tasks_completed_30d: 2,
        tasks_in_progress: 1,
        avg_completion_days: 3,
      },
    });
  });

  it('支持按角色和关键字重新拉取用户列表', async () => {
    render(<UserManagementPage />);

    expect(await screen.findByText('人员管理')).toBeInTheDocument();

    fireEvent.change(screen.getAllByRole('combobox')[0], {
      target: { value: 'developer' },
    });

    await waitFor(() => {
      expect(mockedUserManagementService.getUsers).toHaveBeenCalledWith({
        role: 'developer',
        search: undefined,
      });
    });

    await waitFor(() => {
      expect(screen.getByPlaceholderText('搜索邮箱或用户名...')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByPlaceholderText('搜索邮箱或用户名...'), {
      target: { value: 'alice' },
    });

    await waitFor(() => {
      expect(mockedUserManagementService.getUsers).toHaveBeenCalledWith({
        role: 'developer',
        search: 'alice',
      });
    });
  });

  it('创建用户成功后会关闭弹窗并刷新列表', async () => {
    const user = userEvent.setup();

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getByRole('button', { name: '新建用户' }));

    await user.type(screen.getByPlaceholderText('user@example.com'), 'new@example.com');
    await user.type(screen.getByPlaceholderText('用户名'), 'newuser');
    await user.type(screen.getByPlaceholderText('至少6位'), 'password123');
    await user.click(screen.getByRole('button', { name: '创建' }));

    await waitFor(() => {
      expect(mockedUserManagementService.createUser).toHaveBeenCalledWith({
        email: 'new@example.com',
        username: 'newuser',
        password: 'password123',
        role: 'developer',
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('用户创建成功');
    await waitFor(() => {
      expect(mockedUserManagementService.getUsers).toHaveBeenCalledTimes(2);
    });
    expect(screen.queryByPlaceholderText('user@example.com')).not.toBeInTheDocument();
  });

  it('删除用户后会调用接口并刷新列表', async () => {
    const user = userEvent.setup();

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getAllByRole('button', { name: '删除' })[0]);
    await user.click(screen.getByRole('button', { name: '确认删除' }));

    await waitFor(() => {
      expect(mockedUserManagementService.deleteUser).toHaveBeenCalledWith(1);
    });

    expect(showSuccess).toHaveBeenCalledWith('用户删除成功');
    await waitFor(() => {
      expect(mockedUserManagementService.getUsers).toHaveBeenCalledTimes(2);
    });
  });

  it('可以查看用户工作负载详情', async () => {
    const user = userEvent.setup();

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getAllByRole('button', { name: '负载' })[0]);

    await waitFor(() => {
      expect(mockedUserManagementService.getUserWorkload).toHaveBeenCalledWith(1);
    });

    expect(await screen.findByText('用户工作负载')).toBeInTheDocument();
    expect(screen.getByText(/登录故事/)).toBeInTheDocument();
    expect(screen.getByText(/实现接口/)).toBeInTheDocument();
  });

  it('空用户列表时会展示空状态', async () => {
    mockedUserManagementService.getUsers.mockResolvedValueOnce({
      users: [],
      total: 0,
      page: 1,
      limit: 20,
    });

    render(<UserManagementPage />);

    expect(await screen.findByText('人员管理')).toBeInTheDocument();
    expect(screen.getAllByText('暂无用户')).toHaveLength(2);
  });

  it('编辑用户成功时只提交变更字段并刷新列表', async () => {
    const user = userEvent.setup();

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getAllByRole('button', { name: '编辑' })[0]);

    const usernameInput = screen.getByPlaceholderText('用户名');
    await user.clear(usernameInput);
    await user.type(usernameInput, 'alice');

    const roleSelect = screen.getAllByRole('combobox').at(-1);
    if (!(roleSelect instanceof HTMLSelectElement)) {
      throw new Error('edit role select not found');
    }
    await user.selectOptions(roleSelect, 'tester');
    await user.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => {
      expect(mockedUserManagementService.updateUser).toHaveBeenCalledWith(1, {
        username: 'alice',
        role: 'tester',
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('用户更新成功');
    await waitFor(() => {
      expect(mockedUserManagementService.getUsers).toHaveBeenCalledTimes(2);
    });
  });

  it('创建用户失败时会展示错误提示并保留弹窗', async () => {
    const user = userEvent.setup();
    mockedUserManagementService.createUser.mockRejectedValueOnce(new Error('邮箱已存在'));

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getByRole('button', { name: '新建用户' }));

    await user.type(screen.getByPlaceholderText('user@example.com'), 'new@example.com');
    await user.type(screen.getByPlaceholderText('用户名'), 'newuser');
    await user.type(screen.getByPlaceholderText('至少6位'), 'password123');
    await user.click(screen.getByRole('button', { name: '创建' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('邮箱已存在');
    });

    expect(screen.getByPlaceholderText('user@example.com')).toBeInTheDocument();
  });

  it('工作负载加载失败时会关闭弹窗并提示错误', async () => {
    const user = userEvent.setup();
    mockedUserManagementService.getUserWorkload.mockRejectedValueOnce(
      new Error('工作负载接口失败')
    );

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getAllByRole('button', { name: '负载' })[0]);

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('工作负载接口失败');
    });
    await waitFor(() => {
      expect(screen.queryByText('用户工作负载')).not.toBeInTheDocument();
    });
  });

  it('删除用户失败时会保留确认框并提示错误', async () => {
    const user = userEvent.setup();
    mockedUserManagementService.deleteUser.mockRejectedValueOnce(
      new Error('不能删除最后一个管理员')
    );

    render(<UserManagementPage />);

    await screen.findByText('人员管理');
    await user.click(screen.getAllByRole('button', { name: '删除' })[0]);
    await user.click(screen.getByRole('button', { name: '确认删除' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('不能删除最后一个管理员');
    });

    expect(screen.getByText('删除用户')).toBeInTheDocument();
  });
});
