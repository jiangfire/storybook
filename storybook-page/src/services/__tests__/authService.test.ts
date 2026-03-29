import api from '../api';
import { authService } from '../authService';

vi.mock('../api', () => ({
  default: {
    post: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('authService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('login/register/refreshToken 应命中认证接口', async () => {
    mockedApi.post
      .mockResolvedValueOnce({ data: { data: { token: 'a', user: { id: 1 }, expires_at: 'x' } } })
      .mockResolvedValueOnce({ data: { data: { token: 'b', user: { id: 2 }, expires_at: 'y' } } })
      .mockResolvedValueOnce({ data: { data: { token: 'c', expires_at: 'z' } } });

    await authService.login({ email: 'a@example.com', password: 'secret' });
    await authService.register({
      email: 'b@example.com',
      password: 'secret',
      role: 'developer',
    });
    await authService.refreshToken({ refresh_token: 'refresh' });

    expect(mockedApi.post).toHaveBeenNthCalledWith(1, '/api/auth/login', {
      email: 'a@example.com',
      password: 'secret',
    });
    expect(mockedApi.post).toHaveBeenNthCalledWith(2, '/api/auth/register', {
      email: 'b@example.com',
      password: 'secret',
      role: 'developer',
    });
    expect(mockedApi.post).toHaveBeenNthCalledWith(3, '/api/auth/refresh', {
      refresh_token: 'refresh',
    });
  });

  it('logout 会清理本地 token', () => {
    localStorage.setItem('token', 'token');
    localStorage.setItem('refresh_token', 'refresh');

    authService.logout();

    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('refresh_token')).toBeNull();
  });
});
