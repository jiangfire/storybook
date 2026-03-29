import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useAuthStore } from '../../../stores/authStore';
import RegisterPage from '../RegisterPage';

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
    <MemoryRouter initialEntries={['/register']}>
      <Routes>
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/projects" element={<div>Projects Page</div>} />
        <Route path="/login" element={<div>Login Page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    setAuthStore();
  });

  it('提交时会校验确认密码是否一致', async () => {
    const user = userEvent.setup();
    const register = vi.fn(async () => {});
    const clearError = vi.fn();

    setAuthStore({ register, clearError });

    renderPage();

    await user.type(screen.getByLabelText('邮箱'), 'tester@example.com');
    await user.type(screen.getByLabelText('密码'), 'password123');
    await user.type(screen.getByLabelText('确认密码'), 'password456');
    await user.click(screen.getByRole('button', { name: '注册' }));

    expect(clearError).toHaveBeenCalled();
    expect(await screen.findByText('两次输入的密码不一致')).toBeInTheDocument();
    expect(register).not.toHaveBeenCalled();
  });

  it('切换角色后会按所选角色注册并跳转到项目列表', async () => {
    const user = userEvent.setup();
    const register = vi.fn(async () => {
      useAuthStore.setState({
        user: {
          id: 3,
          email: 'pm@example.com',
          role: 'product',
          created_at: NOW,
        },
        token: 'token',
        isAuthenticated: true,
      });
    });

    setAuthStore({ register });

    renderPage();

    await user.type(screen.getByLabelText('邮箱'), 'pm@example.com');
    await user.click(screen.getByRole('button', { name: /产品经理/ }));
    await user.type(screen.getByLabelText('密码'), 'password123');
    await user.type(screen.getByLabelText('确认密码'), 'password123');
    await user.click(screen.getByRole('button', { name: '注册' }));

    await waitFor(() => {
      expect(register).toHaveBeenCalledWith({
        email: 'pm@example.com',
        password: 'password123',
        role: 'product',
      });
    });

    expect(await screen.findByText('Projects Page')).toBeInTheDocument();
  });

  it('会展示 store 中的注册错误', () => {
    setAuthStore({ error: '邮箱已被使用' });

    renderPage();

    expect(screen.getByText('邮箱已被使用')).toBeInTheDocument();
  });
});
