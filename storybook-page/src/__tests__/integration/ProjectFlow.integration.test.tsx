import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { describe, it, expect, beforeEach } from 'vitest';
import ProjectListPage from '../../pages/Projects/ProjectListPage';
import { useAuthStore } from '../../stores/authStore';
import { useProjectStore } from '../../stores/projectStore';
import { ToastProvider } from '../../components/ui/Toast';

function TestWrapper({ children }: { children: React.ReactNode }) {
  return (
    <ToastProvider>
      {children}
    </ToastProvider>
  );
}

function renderWithRouter(ui: React.ReactElement, initialEntries = ['/projects']) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <Routes>
        <Route
          path="/projects"
          element={<TestWrapper>{ui}</TestWrapper>}
        />
        <Route path="/projects/:id" element={<div data-testid="project-detail">Project Detail</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('Project Flow Integration', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      user: { id: 1, email: 'test@example.com', username: 'tester', role: 'developer', created_at: '', updated_at: '' },
      token: 'mock-token',
      isAuthenticated: true,
      isLoading: false,
      error: null,
    });
    useProjectStore.setState({
      projects: [],
      isLoading: false,
      error: null,
    });
  });

  it('应能创建项目并显示在项目列表中', async () => {
    const user = userEvent.setup();
    renderWithRouter(<ProjectListPage />);

    await waitFor(() => {
      expect(screen.getByText('全部项目')).toBeInTheDocument();
    });

    const createButtons = screen.getAllByRole('button', { name: /新建项目/ });
    await user.click(createButtons[0]);
    await user.type(screen.getByPlaceholderText('例如：电商平台'), '我的测试项目');
    await user.type(screen.getByPlaceholderText('简要描述项目的目标和范围...'), '这是一个集成测试项目');

    await user.click(screen.getByRole('button', { name: /创建项目/ }));

    await waitFor(() => {
      expect(screen.getByTestId('project-detail')).toBeInTheDocument();
    });
  });

  it('项目名称过短时应在表单中显示错误', async () => {
    const user = userEvent.setup();
    renderWithRouter(<ProjectListPage />);

    await waitFor(() => {
      expect(screen.getByText('全部项目')).toBeInTheDocument();
    });

    const createButtons = screen.getAllByRole('button', { name: /新建项目/ });
    await user.click(createButtons[0]);
    await user.type(screen.getByPlaceholderText('例如：电商平台'), 'A');
    await user.click(screen.getByRole('button', { name: /创建项目/ }));

    await waitFor(() => {
      expect(screen.getByText(/项目名称长度应在/)).toBeInTheDocument();
    });
  });
});
