import api from '../api';
import { techLeadService } from '../techLeadService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('techLeadService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getPendingStories 应请求正确接口并返回 data.data', async () => {
    mockedApi.get.mockResolvedValueOnce({
      data: { data: { stories: [], total: 0, page: 1, limit: 20 } },
    });

    const result = await techLeadService.getPendingStories({ page: 1 });

    expect(mockedApi.get).toHaveBeenCalledWith('/api/techlead/pending-stories', {
      params: { page: 1 },
    });
    expect(result.total).toBe(0);
  });

  it('reviewStory 应请求故事审批接口', async () => {
    mockedApi.post.mockResolvedValueOnce({ data: { data: { ok: true } } });

    await techLeadService.reviewStory(12, { approved: false, comment: '原因' });

    expect(mockedApi.post).toHaveBeenCalledWith('/api/stories/12/review', {
      approved: false,
      comment: '原因',
    });
  });

  it('assignStory/getWorkload/getMyProjects 应命中对应接口', async () => {
    mockedApi.patch.mockResolvedValueOnce({ data: { data: { ok: true } } });
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { workloads: [] } } })
      .mockResolvedValueOnce({ data: { data: { projects: [] } } });

    await techLeadService.assignStory(3, 99);
    const workload = await techLeadService.getWorkload(7);
    const projects = await techLeadService.getMyProjects();

    expect(mockedApi.patch).toHaveBeenCalledWith('/api/stories/3/assignee', {
      assigned_to: 99,
    });
    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/techlead/workload', {
      params: { project_id: 7 },
    });
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/techlead/projects');
    expect(workload.workloads).toEqual([]);
    expect(projects.projects).toEqual([]);
  });

  it('techLead 管理接口调用正确', async () => {
    mockedApi.get.mockResolvedValueOnce({ data: { data: { tech_leads: [] } } });
    mockedApi.post.mockResolvedValueOnce({ data: { data: { ok: true } } });
    mockedApi.delete.mockResolvedValueOnce({ data: { data: { ok: true } } });

    await techLeadService.getProjectTechLeads(1);
    await techLeadService.addTechLead(1, 2);
    await techLeadService.removeTechLead(1, 2);

    expect(mockedApi.get).toHaveBeenCalledWith('/api/projects/1/techleads');
    expect(mockedApi.post).toHaveBeenCalledWith('/api/projects/1/techleads', { user_id: 2 });
    expect(mockedApi.delete).toHaveBeenCalledWith('/api/projects/1/techleads/2');
  });
});
