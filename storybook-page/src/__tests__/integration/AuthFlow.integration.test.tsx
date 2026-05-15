import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { describe, it, expect, beforeEach } from 'vitest';
import LoginPage from '../../pages/Auth/LoginPage';
import RegisterPage from '../../pages/Auth/RegisterPage';
import { useAuthStore } from '../../stores/authStore';

function renderWithRouter(ui: React.ReactElement, initialEntries = ['/']) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <Routes>
        <Route path="/" element={ui} />
        <Route path="/projects" element={<div data-testid="projects-page">Projects</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('Auth Flow Integration', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
    });
  });

  it('用户注册成功后应重定向到项目页', async () => {
    const user = userEvent.setup();
    renderWithRouter(<RegisterPage />, ['/']);

    await user.type(screen.getByLabelText('邮箱'), 'test@example.com');
    await user.type(screen.getByLabelText('密码'), 'Password123');
    await user.type(screen.getByLabelText('确认密码'), 'Password123');
    await user.click(screen.getByText('开发人员'));
    await user.click(screen.getByRole('button', { name: /注册/ }));

    await waitFor(() => {
      expect(screen.getByTestId('projects-page')).toBeInTheDocument();
    });
    expect(localStorage.getItem('token')).toBeTruthy();
  });

  it('用户登录成功后应重定向到项目页', async () => {
    const user = userEvent.setup();
    renderWithRouter(<LoginPage />, ['/']);

    await user.type(screen.getByLabelText('邮箱'), 'test@example.com');
    await user.type(screen.getByLabelText('密码'), 'Password123');
    await user.click(screen.getByRole('button', { name: /登录/ }));

    await waitFor(() => {
      expect(screen.getByTestId('projects-page')).toBeInTheDocument();
    });
    expect(localStorage.getItem('token')).toBeTruthy();
  });

  it('邮箱格式错误时应显示验证提示', async () => {
    const user = userEvent.setup();
    renderWithRouter(<LoginPage />, ['/']);

    await user.type(screen.getByLabelText('邮箱'), 'invalid-email');
    await user.type(screen.getByLabelText('密码'), 'Password123');
    await user.click(screen.getByRole('button', { name: /登录/ }));

    await waitFor(() => {
      expect(screen.getByText('邮箱格式不正确')).toBeInTheDocument();
    });
  });
});
