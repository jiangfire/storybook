import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ProtectedRoute from '../ProtectedRoute';
import { useAuthStore } from '../../stores/authStore';

function renderWithRouter(initialEntry = '/private') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="/private"
          element={
            <ProtectedRoute>
              <div>私有内容</div>
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin"
          element={
            <ProtectedRoute allowedRoles={['tech_lead', 'admin']}>
              <div>管理内容</div>
            </ProtectedRoute>
          }
        />
        <Route path="/dashboard" element={<div>工作台</div>} />
        <Route path="/login" element={<div>登录页</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('ProtectedRoute', () => {
  beforeEach(() => {
    localStorage.removeItem('auth-storage');
    useAuthStore.setState({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
    });
  });

  it('未登录时跳转到登录页', () => {
    renderWithRouter();
    expect(screen.getByText('登录页')).toBeInTheDocument();
  });

  it('已登录时展示子内容', () => {
    useAuthStore.setState({
      isAuthenticated: true,
      token: 'token',
      user: {
        id: 1,
        email: 'demo@example.com',
        role: 'developer',
        created_at: '2026-03-08T00:00:00Z',
      },
    });

    renderWithRouter();
    expect(screen.getByText('私有内容')).toBeInTheDocument();
  });

  it('角色不匹配时跳转到工作台', () => {
    useAuthStore.setState({
      isAuthenticated: true,
      token: 'token',
      user: {
        id: 1,
        email: 'dev@example.com',
        role: 'developer',
        created_at: '2026-03-08T00:00:00Z',
      },
    });

    renderWithRouter('/admin');
    expect(screen.getByText('工作台')).toBeInTheDocument();
  });

  it('角色匹配时展示受限内容', () => {
    useAuthStore.setState({
      isAuthenticated: true,
      token: 'token',
      user: {
        id: 2,
        email: 'lead@example.com',
        role: 'tech_lead',
        created_at: '2026-03-08T00:00:00Z',
      },
    });

    renderWithRouter('/admin');
    expect(screen.getByText('管理内容')).toBeInTheDocument();
  });
});
