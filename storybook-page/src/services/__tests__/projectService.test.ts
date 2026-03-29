import api from '../api';
import { projectService } from '../projectService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('projectService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getProjectMembers 和 getProjectMemberCandidates 应命中成员接口', async () => {
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { members: [] } } })
      .mockResolvedValueOnce({ data: { data: { users: [] } } });

    await projectService.getProjectMembers(1);
    await projectService.getProjectMemberCandidates(1);

    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/projects/1/members');
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/projects/1/member-candidates');
  });

  it('createSprint / updateSprintStatus / reports 接口路径正确', async () => {
    mockedApi.post.mockResolvedValueOnce({ data: { data: { id: 7 } } });
    mockedApi.patch.mockResolvedValueOnce({ data: { data: { ok: true } } });
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { sprints: [] } } })
      .mockResolvedValueOnce({
        data: { data: { project_id: 1, sprint: { id: 7 }, baseline_points: 0, points: [] } },
      })
      .mockResolvedValueOnce({ data: { data: { project_id: 1, velocity: [] } } })
      .mockResolvedValueOnce({
        data: {
          data: {
            project_id: 1,
            bugs: { total: 0, status_breakdown: {}, severity_breakdown: {} },
            acceptance_criteria: { total: 0, passed: 0, failed: 0, completion_percentage: 0 },
          },
        },
      });

    await projectService.createSprint(1, {
      name: 'Sprint 1',
      start_date: '2026-03-29',
      end_date: '2026-04-05',
    });
    await projectService.updateSprintStatus(7, { status: 'active' });
    await projectService.getSprints(1);
    await projectService.getBurndown(1, 7);
    await projectService.getVelocity(1);
    await projectService.getQuality(1);

    expect(mockedApi.post).toHaveBeenCalledWith('/api/projects/1/sprints', {
      name: 'Sprint 1',
      start_date: '2026-03-29',
      end_date: '2026-04-05',
    });
    expect(mockedApi.patch).toHaveBeenCalledWith('/api/sprints/7/status', { status: 'active' });
    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/projects/1/sprints');
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/projects/1/reports/burndown', {
      params: { sprint_id: 7 },
    });
    expect(mockedApi.get).toHaveBeenNthCalledWith(3, '/api/projects/1/reports/velocity');
    expect(mockedApi.get).toHaveBeenNthCalledWith(4, '/api/projects/1/reports/quality');
  });
});
