import { projectService } from '../../services/projectService';
import { useProjectStore } from '../projectStore';

vi.mock('../../services/projectService', () => ({
  projectService: {
    getProjects: vi.fn(),
    getProject: vi.fn(),
    getProjectOverview: vi.fn(),
    createProject: vi.fn(),
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
  },
}));

const mockedProjectService = vi.mocked(projectService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function resetStore() {
  useProjectStore.setState({
    currentProject: null,
    projects: [],
    projectOverview: null,
    isLoading: false,
    error: null,
  });
}

describe('projectStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it('fetchProject 与 fetchProjectOverview 会更新当前项目和概览', async () => {
    mockedProjectService.getProject.mockResolvedValue({
      id: 1,
      name: 'Alpha',
      agile_mode: 'kanban',
      owner: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: NOW,
      },
      created_at: NOW,
    });
    mockedProjectService.getProjectOverview.mockResolvedValue({
      project: { id: 1, name: 'Alpha' },
      statistics: {
        total_stories: 1,
        status_breakdown: {
          pending: 1,
          backlog: 0,
          ready: 0,
          in_progress: 0,
          test: 0,
          done: 0,
        },
        completion_rate: 0,
        active_members: 1,
        avg_story_points: 3,
      },
      recent_activities: [],
    });

    await useProjectStore.getState().fetchProject(1);
    await useProjectStore.getState().fetchProjectOverview(1);

    expect(useProjectStore.getState().currentProject?.name).toBe('Alpha');
    expect(useProjectStore.getState().projectOverview?.statistics.total_stories).toBe(1);
  });

  it('create/update/delete 会同步维护 projects 与 currentProject', async () => {
    mockedProjectService.createProject.mockResolvedValue({
      id: 1,
      name: 'Alpha',
      agile_mode: 'kanban',
      owner: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: NOW,
      },
      created_at: NOW,
    });
    mockedProjectService.updateProject.mockResolvedValue({
      id: 1,
      name: 'Alpha v2',
      agile_mode: 'kanban',
      owner: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: NOW,
      },
      created_at: NOW,
    });
    mockedProjectService.deleteProject.mockResolvedValue(undefined);

    const created = await useProjectStore.getState().createProject({
      name: 'Alpha',
      agile_mode: 'kanban',
    });
    useProjectStore.getState().setCurrentProject(created);
    await useProjectStore.getState().updateProject(1, { name: 'Alpha v2' });
    await useProjectStore.getState().deleteProject(1);

    expect(useProjectStore.getState().projects).toEqual([]);
    expect(useProjectStore.getState().currentProject).toBeNull();
  });

  it('fetchProjects 失败时会记录错误', async () => {
    mockedProjectService.getProjects.mockRejectedValue(new Error('项目列表加载失败'));

    await useProjectStore.getState().fetchProjects();

    expect(useProjectStore.getState().error).toBe('项目列表加载失败');
    expect(useProjectStore.getState().isLoading).toBe(false);
  });
});
