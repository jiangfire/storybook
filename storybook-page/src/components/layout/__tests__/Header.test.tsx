import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { User } from '../../../types/models';
import { searchService } from '../../../services/searchService';
import { useAuthStore } from '../../../stores/authStore';
import Header from '../Header';

const navigate = vi.fn();

vi.mock('../../../services/searchService', () => ({
  searchService: {
    getCapabilities: vi.fn(),
    search: vi.fn(),
    searchSemanticStories: vi.fn(),
  },
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => navigate,
  };
});

vi.mock('../../../hooks/useWebSocket', () => ({
  useWebSocket: () => ({ isConnected: true }),
}));

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    showInfo: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showWarning: vi.fn(),
    showToast: vi.fn(),
  }),
}));

vi.mock('../../../stores/notificationStore', () => ({
  useNotificationStore: () => ({
    items: [],
    unreadCount: 0,
    isLoading: false,
    error: null,
    fetchList: vi.fn(),
    fetchUnreadCount: vi.fn(),
    markRead: vi.fn(),
    markAllRead: vi.fn(),
    prepend: vi.fn(),
    clearError: vi.fn(),
  }),
}));

vi.mock('../../notifications/NotificationBell', () => ({
  default: () => <button type="button" aria-label="通知" />,
}));

const mockedSearchService = vi.mocked(searchService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function setAuthUser(role: User['role'], logout = vi.fn()) {
  useAuthStore.setState({
    user: {
      id: 1,
      email: `${role}@example.com`,
      role,
      created_at: NOW,
    },
    token: 'token',
    isAuthenticated: true,
    isLoading: false,
    error: null,
    logout,
  });
  return logout;
}

describe('Header', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockedSearchService.getCapabilities.mockResolvedValue({ semantic_enabled: true });
    mockedSearchService.search.mockResolvedValue({
      projects: [
        {
          id: 1,
          name: 'Alpha',
          description: '认证项目',
          agile_mode: 'kanban',
          created_at: NOW,
        },
      ],
      stories: [
        {
          id: 18,
          project_id: 1,
          title: '登录故事',
          story_type: 'feature',
          status: 'pending',
          priority: 2,
          updated_at: NOW,
        },
      ],
      bugs: [
        {
          id: 9,
          project_id: 1,
          title: '登录缺陷',
          severity: 'high',
          status: 'open',
          updated_at: NOW,
        },
      ],
    });
    mockedSearchService.searchSemanticStories.mockResolvedValue({
      stories: [
        {
          id: 18,
          project_id: 1,
          title: '登录故事',
          story_type: 'feature',
          status: 'pending',
          priority: 2,
          similarity: 0.87,
        },
      ],
    });
  });

  it('会加载搜索能力并支持退出登录', async () => {
    const user = userEvent.setup();
    const logout = setAuthUser('product');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(mockedSearchService.getCapabilities).toHaveBeenCalledTimes(1);
    });

    expect(screen.getByLabelText('语义搜索已启用')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /product@example\.com/i }));
    await user.click(screen.getByRole('button', { name: '退出登录' }));

    expect(logout).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith('/login');
  });

  it('输入关键字后会展示搜索结果并支持跳转', async () => {
    const user = userEvent.setup();
    setAuthUser('admin');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    await user.type(screen.getByPlaceholderText('搜索项目 / 故事 / 缺陷'), 'login');

    await waitFor(() => {
      expect(mockedSearchService.search).toHaveBeenCalledWith({
        q: 'login',
        type: 'all',
        limit: 8,
      });
      expect(mockedSearchService.searchSemanticStories).toHaveBeenCalledWith('login', 5);
    });

    expect(await screen.findByText('Alpha')).toBeInTheDocument();
    expect(screen.getByText('87%')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '#9 登录缺陷' }));

    expect(navigate).toHaveBeenCalledWith('/projects/1/bugs?bug=9');
  });

  it('搜索能力探测失败时会降级显示未启用状态', async () => {
    mockedSearchService.getCapabilities.mockRejectedValueOnce(new Error('capabilities offline'));
    setAuthUser('developer');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    expect(await screen.findByLabelText('语义搜索未启用')).toBeInTheDocument();
  });

  it('关键字搜索失败时会展示错误信息', async () => {
    mockedSearchService.search.mockRejectedValueOnce(new Error('搜索接口失败'));
    setAuthUser('admin');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    fireEvent.change(screen.getByPlaceholderText('搜索项目 / 故事 / 缺陷'), {
      target: { value: 'login' },
    });

    await waitFor(() => {
      expect(mockedSearchService.search).toHaveBeenCalledWith({
        q: 'login',
        type: 'all',
        limit: 8,
      });
    });

    expect(await screen.findByText('搜索接口失败')).toBeInTheDocument();
  });

  it('语义搜索异常时会展示降级占位，并支持 Escape 关闭结果', async () => {
    mockedSearchService.search.mockResolvedValueOnce({
      projects: [],
      stories: [],
      bugs: [],
    });
    mockedSearchService.searchSemanticStories.mockRejectedValueOnce(new Error('语义查询异常'));
    setAuthUser('admin');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    const input = screen.getByPlaceholderText('搜索项目 / 故事 / 缺陷');
    fireEvent.change(input, {
      target: { value: 'login' },
    });

    expect(await screen.findByLabelText('语义查询异常')).toBeInTheDocument();

    fireEvent.keyDown(input, { key: 'Escape' });

    await waitFor(() => {
      expect(screen.queryByLabelText('语义查询异常')).not.toBeInTheDocument();
    });
  });

  it('点击外部区域会同时收起搜索结果和用户菜单', async () => {
    const user = userEvent.setup();
    setAuthUser('product');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    fireEvent.change(screen.getByPlaceholderText('搜索项目 / 故事 / 缺陷'), {
      target: { value: 'login' },
    });

    expect(await screen.findByText('Alpha')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /product@example\.com/i }));
    expect(screen.getByText('个人工作台')).toBeInTheDocument();

    fireEvent.mouseDown(document.body);

    await waitFor(() => {
      expect(screen.queryByText('Alpha')).not.toBeInTheDocument();
      expect(screen.queryByText('个人工作台')).not.toBeInTheDocument();
    });
  });

  it('没有可回退历史时会回到项目列表', async () => {
    const historyLengthSpy = vi.spyOn(window.history, 'length', 'get').mockReturnValue(1);
    const user = userEvent.setup();
    setAuthUser('developer');

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>
    );

    await user.click(screen.getByRole('button', { name: '返回上一页' }));

    expect(navigate).toHaveBeenCalledWith('/projects');
    historyLengthSpy.mockRestore();
  });
});
