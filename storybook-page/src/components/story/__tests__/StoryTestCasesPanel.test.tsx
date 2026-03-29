import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import { testCaseService } from '../../../services/testCaseService';
import StoryTestCasesPanel from '../StoryTestCasesPanel';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/testCaseService', () => ({
  testCaseService: {
    getStoryTestCases: vi.fn(),
    createTestCase: vi.fn(),
    updateTestCaseStatus: vi.fn(),
  },
}));

const mockedTestCaseService = vi.mocked(testCaseService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

const baseCase = {
  id: 71,
  story_id: 33,
  title: '验证登录成功',
  description: '输入合法账号密码',
  steps: ['打开登录页', '输入账号密码'],
  expected_result: '进入首页',
  status: 'pending' as const,
  created_at: NOW,
  updated_at: NOW,
};

function setAuthUser(role: User['role']) {
  useAuthStore.setState({
    user: {
      id: 9,
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

describe('StoryTestCasesPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();

    mockedTestCaseService.getStoryTestCases.mockResolvedValue({
      test_cases: [baseCase],
    });
    mockedTestCaseService.createTestCase.mockResolvedValue(baseCase);
    mockedTestCaseService.updateTestCaseStatus.mockResolvedValue({
      ...baseCase,
      status: 'passed',
    });
  });

  it('tester 可以创建测试用例，并按行拆分步骤', async () => {
    const user = userEvent.setup();
    setAuthUser('tester');

    render(<StoryTestCasesPanel storyId={33} />);

    await waitFor(() => {
      expect(screen.getByText('验证登录成功')).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText('测试用例标题'), ' 登录成功 ');
    await user.type(screen.getByPlaceholderText('测试步骤（每行一条）'), ' 打开登录页 \n 输入账号密码 \n');
    await user.type(screen.getByPlaceholderText('预期结果（可选）'), '进入首页');
    await user.click(screen.getByRole('button', { name: '新建测试用例' }));

    await waitFor(() => {
      expect(mockedTestCaseService.createTestCase).toHaveBeenCalledWith(33, {
        title: '登录成功',
        description: undefined,
        steps: ['打开登录页', '输入账号密码'],
        expected_result: '进入首页',
      });
    });

    await user.selectOptions(screen.getByRole('combobox'), 'passed');

    await waitFor(() => {
      expect(mockedTestCaseService.updateTestCaseStatus).toHaveBeenCalledWith(71, {
        status: 'passed',
      });
    });
  });

  it('非 tester/admin 只能看到只读状态，不显示创建入口', async () => {
    setAuthUser('developer');

    render(<StoryTestCasesPanel storyId={33} />);

    await waitFor(() => {
      expect(screen.getByText('验证登录成功')).toBeInTheDocument();
    });

    expect(screen.queryByRole('button', { name: '新建测试用例' })).not.toBeInTheDocument();
    expect(screen.queryByPlaceholderText('测试步骤（每行一条）')).not.toBeInTheDocument();
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.getByText('待验证')).toBeInTheDocument();
  });

  it('admin 可以刷新列表并更新测试状态', async () => {
    const user = userEvent.setup();
    setAuthUser('admin');

    render(<StoryTestCasesPanel storyId={33} />);

    await screen.findByText('验证登录成功');
    await user.click(screen.getByRole('button', { name: '刷新' }));

    await waitFor(() => {
      expect(mockedTestCaseService.getStoryTestCases).toHaveBeenCalledTimes(2);
    });

    await user.selectOptions(screen.getByRole('combobox'), 'failed');

    await waitFor(() => {
      expect(mockedTestCaseService.updateTestCaseStatus).toHaveBeenCalledWith(71, {
        status: 'failed',
      });
    });
  });

  it('加载失败和表单校验失败时会展示错误信息', async () => {
    const user = userEvent.setup();
    mockedTestCaseService.getStoryTestCases.mockRejectedValueOnce(new Error('测试用例接口异常'));
    setAuthUser('tester');

    render(<StoryTestCasesPanel storyId={33} />);

    expect(await screen.findByText('测试用例接口异常')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '新建测试用例' }));
    expect(showError).toHaveBeenCalledWith('请输入测试用例标题');

    await user.type(screen.getByPlaceholderText('测试用例标题'), '边界校验');
    await user.click(screen.getByRole('button', { name: '新建测试用例' }));

    expect(showError).toHaveBeenCalledWith('至少输入一个测试步骤（每行一条）');
  });
});
