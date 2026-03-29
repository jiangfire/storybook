import { authService } from '../../services/authService';
import { useAuthStore } from '../authStore';

vi.mock('../../services/authService', () => ({
  authService: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  },
}));

const mockedAuthService = vi.mocked(authService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function resetStore() {
  useAuthStore.setState({
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
  });
}

describe('authStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    resetStore();
  });

  it('login 成功后会写入 token 并更新鉴权状态', async () => {
    mockedAuthService.login.mockResolvedValue({
      user: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: NOW,
      },
      token: 'access-token',
      refresh_token: 'refresh-token',
      expires_at: NOW,
    });

    await useAuthStore.getState().login({ email: 'pm@example.com', password: 'secret' });

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.token).toBe('access-token');
    expect(localStorage.getItem('token')).toBe('access-token');
    expect(localStorage.getItem('refresh_token')).toBe('refresh-token');
  });

  it('register 失败时会保留错误并继续抛出异常', async () => {
    const error = {
      response: { data: { message: '注册失败' } },
    };
    mockedAuthService.register.mockRejectedValue(error);

    await expect(
      useAuthStore.getState().register({
        email: 'dev@example.com',
        password: 'secret',
        role: 'developer',
      })
    ).rejects.toEqual(error);

    expect(useAuthStore.getState().error).toBe('注册失败');
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it('logout 会调用服务并清空当前用户状态', () => {
    useAuthStore.setState({
      user: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: NOW,
      },
      token: 'token',
      isAuthenticated: true,
      isLoading: false,
      error: null,
    });

    useAuthStore.getState().logout();

    expect(mockedAuthService.logout).toHaveBeenCalled();
    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().token).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it('login 失败时会保留错误并结束 loading 状态', async () => {
    const error = new Error('账号或密码错误');
    mockedAuthService.login.mockRejectedValue(error);

    await expect(
      useAuthStore.getState().login({ email: 'pm@example.com', password: 'wrong' })
    ).rejects.toEqual(error);

    const state = useAuthStore.getState();
    expect(state.error).toBe('账号或密码错误');
    expect(state.isLoading).toBe(false);
    expect(state.isAuthenticated).toBe(false);
  });

  it('register 成功但没有 refresh token 时不会写入 refresh_token', async () => {
    mockedAuthService.register.mockResolvedValue({
      user: {
        id: 2,
        email: 'tester@example.com',
        role: 'tester',
        created_at: NOW,
      },
      token: 'register-token',
      expires_at: NOW,
    });

    await useAuthStore.getState().register({
      email: 'tester@example.com',
      password: 'secret',
      role: 'tester',
    });

    expect(useAuthStore.getState().token).toBe('register-token');
    expect(localStorage.getItem('refresh_token')).toBeNull();
  });

  it('clearError 和 setLoading 会更新本地状态', () => {
    useAuthStore.setState({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
      error: '旧错误',
    });

    useAuthStore.getState().clearError();
    expect(useAuthStore.getState().error).toBeNull();

    useAuthStore.getState().setLoading(true);
    expect(useAuthStore.getState().isLoading).toBe(true);
  });
});
