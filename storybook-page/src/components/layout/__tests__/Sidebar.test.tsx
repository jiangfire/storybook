import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { User } from '../../../types/models';
import { useAuthStore } from '../../../stores/authStore';
import Sidebar from '../Sidebar';

const NOW = '2026-03-29T00:00:00Z';

function setAuthUser(role: User['role']) {
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
  });
}

describe('Sidebar', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('admin 能看到技术负责人和管理员菜单', () => {
    setAuthUser('admin');

    render(
      <MemoryRouter initialEntries={['/admin/users']}>
        <Sidebar
          currentProject={{ id: 7, name: 'Alpha', agile_mode: 'scrum' }}
        />
      </MemoryRouter>
    );

    expect(screen.getAllByText('审批').length).toBeGreaterThan(0);
    expect(screen.getAllByText('负载').length).toBeGreaterThan(0);
    expect(screen.getAllByText('人员').length).toBeGreaterThan(0);
    expect(screen.getAllByText('AI').length).toBeGreaterThan(0);
    expect(screen.getAllByText('冲刺模式').length).toBeGreaterThan(0);
  });

  it('developer 不显示受限菜单，但保留项目与工作台入口', () => {
    setAuthUser('developer');

    render(
      <MemoryRouter initialEntries={['/projects']}>
        <Sidebar
          currentProject={{ id: 7, name: 'Alpha', agile_mode: 'kanban' }}
        />
      </MemoryRouter>
    );

    expect(screen.getAllByText('项目').length).toBeGreaterThan(0);
    expect(screen.getAllByText('工作台').length).toBeGreaterThan(0);
    expect(screen.queryByText('AI')).not.toBeInTheDocument();
    expect(screen.queryByText('人员')).not.toBeInTheDocument();
    expect(screen.getAllByText('看板模式').length).toBeGreaterThan(0);
  });
});
