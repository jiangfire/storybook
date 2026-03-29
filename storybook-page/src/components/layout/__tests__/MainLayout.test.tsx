import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useProjectStore } from '../../../stores/projectStore';
import MainLayout from '../MainLayout';

const sidebarSpy = vi.fn();

vi.mock('../Header', () => ({
  default: () => <div>Header Mock</div>,
}));

vi.mock('../Sidebar', () => ({
  default: (props: unknown) => {
    sidebarSpy(props);
    return <div>Sidebar Mock</div>;
  },
}));

function setProjectStore() {
  useProjectStore.setState({
    currentProject: {
      id: 3,
      name: 'Alpha',
      description: '核心项目',
      agile_mode: 'kanban',
      owner: {
        id: 1,
        email: 'owner@example.com',
        role: 'product',
        created_at: '2026-03-29T00:00:00Z',
      },
      created_at: '2026-03-29T00:00:00Z',
      updated_at: '2026-03-29T00:00:00Z',
    },
    projectOverview: null,
    projects: [],
    isLoading: false,
    error: null,
    fetchProjects: vi.fn(async () => {}),
    fetchProject: vi.fn(async () => {}),
    fetchProjectOverview: vi.fn(async () => {}),
    createProject: vi.fn(async () => {
      throw new Error('not implemented');
    }),
    updateProject: vi.fn(async () => {}),
    deleteProject: vi.fn(async () => {}),
    setCurrentProject: vi.fn(),
    clearError: vi.fn(),
  });
}

function renderLayout(showSidebar = true) {
  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <Routes>
        <Route element={<MainLayout showSidebar={showSidebar} />}>
          <Route path="/dashboard" element={<div>Outlet Content</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  );
}

describe('MainLayout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setProjectStore();
  });

  it('默认渲染 Header、Sidebar 和 Outlet', () => {
    renderLayout();

    expect(screen.getByText('Header Mock')).toBeInTheDocument();
    expect(screen.getByText('Sidebar Mock')).toBeInTheDocument();
    expect(screen.getByText('Outlet Content')).toBeInTheDocument();
    expect(sidebarSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        currentProject: expect.objectContaining({ id: 3, name: 'Alpha' }),
      })
    );
  });

  it('showSidebar=false 时不渲染 Sidebar', () => {
    renderLayout(false);

    expect(screen.getByText('Header Mock')).toBeInTheDocument();
    expect(screen.queryByText('Sidebar Mock')).not.toBeInTheDocument();
  });
});
