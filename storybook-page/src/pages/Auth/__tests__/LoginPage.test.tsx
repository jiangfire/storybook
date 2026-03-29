import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useAuthStore } from '../../../stores/authStore';
import LoginPage from '../LoginPage';

const NOW = '2026-03-29T00:00:00Z';

function setAuthStore(overrides: Partial<ReturnType<typeof useAuthStore.getState>> = {}) {
  useAuthStore.setState({
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
    login: vi.fn(async () => {}),
    register: vi.fn(async () => {}),
    logout: vi.fn(),
    clearError: vi.fn(),
    setLoading: vi.fn(),
    ...overrides,
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/login']}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/projects" element={<div>Projects Page</div>} />
        <Route path="/register" element={<div>Register Page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    setAuthStore();
  });

  it('提交时会校验邮箱和密码格式', async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => {});
    const clearError = vi.fn();

    setAuthStore({ login, clearError });

    renderPage();

    await user.type(screen.getByLabelText('邮箱'), 'invalid-email');
    await user.type(screen.getByLabelText('密码'), '123');
    await user.click(screen.getByRole('button', { name: '登录' }));

    expect(clearError).toHaveBeenCalled();
    expect(await screen.findByText('邮箱格式不正确')).toBeInTheDocument();
    expect(screen.getByText('密码至少8位，包含字母和数字')).toBeInTheDocument();
    expect(login).not.toHaveBeenCalled();
  });

  it('登录成功后跳转到项目列表', async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => {
      useAuthStore.setState({
        user: {
          id: 1,
          email: 'owner@example.com',
          role: 'product',
          created_at: NOW,
        },
        token: 'token',
        isAuthenticated: true,
      });
    });

    setAuthStore({ login });

    renderPage();

    await user.type(screen.getByLabelText('邮箱'), 'owner@example.com');
    await user.type(screen.getByLabelText('密码'), 'password123');
    await user.click(screen.getByRole('button', { name: '登录' }));

    await waitFor(() => {
      expect(login).toHaveBeenCalledWith({
        email: 'owner@example.com',
        password: 'password123',
      });
    });

    expect(await screen.findByText('Projects Page')).toBeInTheDocument();
  });

  it('会展示 store 中的登录错误', () => {
    setAuthStore({ error: '账号或密码错误' });

    renderPage();

    expect(screen.getByText('账号或密码错误')).toBeInTheDocument();
  });
});
