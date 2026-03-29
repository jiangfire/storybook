import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import { taskService } from '../../../services/taskService';
import StoryTasksPanel from '../StoryTasksPanel';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/taskService', () => ({
  taskService: {
    getStoryTasks: vi.fn(),
    createTask: vi.fn(),
    splitFromAC: vi.fn(),
    updateTaskStatus: vi.fn(),
    updateTaskProgress: vi.fn(),
    claimTask: vi.fn(),
    releaseTask: vi.fn(),
    deleteTask: vi.fn(),
    getTask: vi.fn(),
    updateTask: vi.fn(),
    addCodeRef: vi.fn(),
  },
}));

const mockedTaskService = vi.mocked(taskService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

const baseTask = {
  id: 11,
  project_id: 1,
  story_id: 21,
  title: '实现登录',
  description: '实现账号密码登录',
  status: 'todo' as const,
  priority: 2,
  progress: 0,
  estimated_hours: 8,
  assigned_to: undefined,
  created_by: {
    id: 1,
    email: 'creator@example.com',
  },
  code_references: [],
  created_at: NOW,
  updated_at: NOW,
};

function setAuthUser(role: User['role'], id: number) {
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

describe('StoryTasksPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();

    mockedTaskService.getStoryTasks.mockResolvedValue({ tasks: [baseTask] });
    mockedTaskService.createTask.mockResolvedValue(baseTask);
    mockedTaskService.getTask.mockResolvedValue(baseTask);
    mockedTaskService.splitFromAC.mockResolvedValue({
      story_id: 21,
      created_count: 2,
      tasks: [baseTask],
    });
    mockedTaskService.claimTask.mockResolvedValue({
      ...baseTask,
      assigned_to: {
        id: 99,
        email: 'developer@example.com',
      },
    });
    mockedTaskService.updateTaskStatus.mockResolvedValue({
      ...baseTask,
      status: 'in_progress',
    });
    mockedTaskService.updateTaskProgress.mockResolvedValue({
      ...baseTask,
      progress: 60,
    });
    mockedTaskService.releaseTask.mockResolvedValue({
      ...baseTask,
      assigned_to: undefined,
      progress: 60,
    });
    mockedTaskService.updateTask.mockResolvedValue(baseTask);
    mockedTaskService.addCodeRef.mockResolvedValue({
      task_id: baseTask.id,
      code_references: ['src/auth.ts:42'],
    });
    mockedTaskService.deleteTask.mockResolvedValue(undefined);
  });

  it('product 可见 AC 自动拆分和新建任务入口，并能触发拆分', async () => {
    const user = userEvent.setup();
    setAuthUser('product', 1);

    render(<StoryTasksPanel storyId={21} />);

    await waitFor(() => {
      expect(screen.getByText('实现登录')).toBeInTheDocument();
    });

    expect(screen.getByRole('button', { name: 'AC自动拆分' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '新建任务' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'AC自动拆分' }));

    await waitFor(() => {
      expect(mockedTaskService.splitFromAC).toHaveBeenCalledWith(21);
      expect(mockedTaskService.getStoryTasks).toHaveBeenCalledTimes(2);
    });
  });

  it('developer 可领取未分配任务，但非创建者不能保存或删除', async () => {
    const user = userEvent.setup();
    setAuthUser('developer', 99);

    render(<StoryTasksPanel storyId={21} />);

    await waitFor(() => {
      expect(screen.getByText('实现登录')).toBeInTheDocument();
    });

    expect(screen.queryByRole('button', { name: 'AC自动拆分' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '删除' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '领取' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '领取' }));

    await waitFor(() => {
      expect(mockedTaskService.claimTask).toHaveBeenCalledWith(11);
    });

    await user.click(screen.getByRole('button', { name: '详情' }));

    await waitFor(() => {
      expect(mockedTaskService.getTask).toHaveBeenCalledWith(11);
    });

    expect(screen.getByDisplayValue('实现登录')).toBeDisabled();
    expect(screen.queryByRole('button', { name: '保存' })).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('例如 src/components/Task.tsx:42')).toBeInTheDocument();
  });

  it('tester 只能查看任务详情，不能创建、领取或添加代码引用', async () => {
    const user = userEvent.setup();
    setAuthUser('tester', 55);

    render(<StoryTasksPanel storyId={21} />);

    await waitFor(() => {
      expect(screen.getByText('实现登录')).toBeInTheDocument();
    });

    expect(screen.queryByRole('button', { name: 'AC自动拆分' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '新建任务' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '领取' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '删除' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '详情' }));

    await waitFor(() => {
      expect(mockedTaskService.getTask).toHaveBeenCalledWith(11);
    });

    expect(screen.queryByPlaceholderText('例如 src/components/Task.tsx:42')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '保存' })).not.toBeInTheDocument();
  });

  it('任务创建者可以新建任务、保存详情、添加代码引用并删除任务', async () => {
    const user = userEvent.setup();
    setAuthUser('admin', 1);
    mockedTaskService.updateTask.mockResolvedValueOnce({
      ...baseTask,
      title: '修复登录链路',
    });

    render(<StoryTasksPanel storyId={21} />);

    await screen.findByText('实现登录');

    await user.type(screen.getByPlaceholderText('任务标题'), ' 修复登录链路 ');
    await user.type(screen.getByPlaceholderText('预估工时'), '6');
    await user.click(screen.getByRole('button', { name: '新建任务' }));

    await waitFor(() => {
      expect(mockedTaskService.createTask).toHaveBeenCalledWith(21, {
        title: '修复登录链路',
        description: undefined,
        priority: 2,
        estimated_hours: 6,
      });
    });

    await user.click(screen.getByRole('button', { name: '详情' }));

    await waitFor(() => {
      expect(mockedTaskService.getTask).toHaveBeenCalledWith(11);
    });

    const titleInput = screen.getByDisplayValue('实现登录');
    await user.clear(titleInput);
    await user.type(titleInput, '修复登录链路');
    await user.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => {
      expect(mockedTaskService.updateTask).toHaveBeenCalledWith(11, {
        title: '修复登录链路',
        description: '实现账号密码登录',
        priority: 2,
        estimated_hours: 8,
      });
    });

    await user.type(screen.getByPlaceholderText('例如 src/components/Task.tsx:42'), 'src/auth.ts:42');
    await user.click(screen.getByRole('button', { name: '添加' }));

    await waitFor(() => {
      expect(mockedTaskService.addCodeRef).toHaveBeenCalledWith(11, 'src/auth.ts:42');
    });

    await user.click(screen.getAllByRole('button', { name: '删除' })[0]);
    await user.click(screen.getByRole('button', { name: '确认删除' }));

    await waitFor(() => {
      expect(mockedTaskService.deleteTask).toHaveBeenCalledWith(11);
    });
  });

  it('列表加载失败时会展示错误，且创建任务会校验标题', async () => {
    const user = userEvent.setup();
    mockedTaskService.getStoryTasks.mockRejectedValueOnce(new Error('任务接口异常'));
    setAuthUser('product', 1);

    render(<StoryTasksPanel storyId={21} />);

    expect(await screen.findByText('任务接口异常')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '新建任务' }));
    expect(showError).toHaveBeenCalledWith('请输入任务标题');
  });

  it('负责人可以更新状态、进度并释放任务', async () => {
    const user = userEvent.setup();
    setAuthUser('admin', 1);
    const assignedTask = {
      ...baseTask,
      assigned_to: {
        id: 99,
        email: 'developer@example.com',
      },
    };
    mockedTaskService.getStoryTasks.mockResolvedValueOnce({
      tasks: [assignedTask],
    });
    mockedTaskService.updateTaskStatus.mockResolvedValueOnce({
      ...assignedTask,
      status: 'in_progress',
    });
    mockedTaskService.updateTaskProgress.mockResolvedValueOnce({
      ...assignedTask,
      status: 'in_progress',
      progress: 60,
    });
    mockedTaskService.releaseTask.mockResolvedValueOnce({
      ...assignedTask,
      status: 'in_progress',
      progress: 60,
      assigned_to: undefined,
    });

    render(<StoryTasksPanel storyId={21} />);

    await screen.findByText('实现登录');

    const statusSelect = screen
      .getAllByRole('combobox')
      .find((element) => element.querySelector('option[value="todo"]'));
    if (!(statusSelect instanceof HTMLSelectElement)) {
      throw new Error('task status select not found');
    }

    await user.selectOptions(statusSelect, 'in_progress');
    await waitFor(() => {
      expect(mockedTaskService.updateTaskStatus).toHaveBeenCalledWith(11, {
        status: 'in_progress',
      });
    });

    const progressInput = document.querySelector('input[max="100"]');
    if (!(progressInput instanceof HTMLInputElement)) {
      throw new Error('progress input not found');
    }
    await user.clear(progressInput);
    await user.type(progressInput, '60');
    expect(mockedTaskService.updateTaskProgress).not.toHaveBeenCalled();
    await user.tab();

    await waitFor(() => {
      expect(mockedTaskService.updateTaskProgress).toHaveBeenCalledWith(11, {
        progress: 60,
      });
    });

    await user.click(screen.getByRole('button', { name: '释放' }));

    await waitFor(() => {
      expect(mockedTaskService.releaseTask).toHaveBeenCalledWith(11);
      expect(showSuccess).toHaveBeenCalledWith('任务已释放');
    });
  });

  it('详情加载失败和保存失败时会提示错误', async () => {
    const user = userEvent.setup();
    setAuthUser('admin', 1);
    mockedTaskService.getTask.mockRejectedValueOnce(new Error('任务详情接口失败'));

    render(<StoryTasksPanel storyId={21} />);

    await screen.findByText('实现登录');
    await user.click(screen.getByRole('button', { name: '详情' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('任务详情接口失败');
    });

    mockedTaskService.getTask.mockResolvedValueOnce(baseTask);
    mockedTaskService.updateTask.mockRejectedValueOnce(new Error('保存失败'));

    await user.click(screen.getByRole('button', { name: '详情' }));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('保存失败');
    });
  });
});
