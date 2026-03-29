import api from '../api';
import { bugService } from '../bugService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('bugService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('应命中缺陷列表、创建、详情、状态和指派接口', async () => {
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { bugs: [] } } })
      .mockResolvedValueOnce({ data: { data: { id: 5 } } });
    mockedApi.post.mockResolvedValueOnce({ data: { data: { id: 5 } } });
    mockedApi.patch
      .mockResolvedValueOnce({ data: { data: { id: 5, status: 'resolved' } } })
      .mockResolvedValueOnce({ data: { data: { id: 5, assigned_to: 9 } } });

    await bugService.getProjectBugs(3, { severity: 'high' });
    await bugService.createBug(3, { title: '缺陷', severity: 'high' });
    await bugService.getBug(5);
    await bugService.updateBugStatus(5, { status: 'resolved' });
    await bugService.assignBug(5, { assigned_to: 9 });

    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/projects/3/bugs', {
      params: { severity: 'high' },
    });
    expect(mockedApi.post).toHaveBeenCalledWith('/api/projects/3/bugs', {
      title: '缺陷',
      severity: 'high',
    });
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/bugs/5');
    expect(mockedApi.patch).toHaveBeenNthCalledWith(1, '/api/bugs/5/status', {
      status: 'resolved',
    });
    expect(mockedApi.patch).toHaveBeenNthCalledWith(2, '/api/bugs/5/assign', {
      assigned_to: 9,
    });
  });
});
