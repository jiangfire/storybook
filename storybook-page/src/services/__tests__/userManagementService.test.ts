import api from '../api';
import { userManagementService } from '../userManagementService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('userManagementService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('应命中用户列表、创建、更新、删除和工作负载接口', async () => {
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { users: [], total: 0, page: 1, limit: 20 } } })
      .mockResolvedValueOnce({
        data: {
          data: { user: { id: 1 }, active_stories: [], active_tasks: [], statistics: {} },
        },
      });
    mockedApi.post.mockResolvedValueOnce({ data: { data: { ok: true } } });
    mockedApi.put.mockResolvedValueOnce({ data: { data: { ok: true } } });
    mockedApi.delete.mockResolvedValueOnce({ data: { data: { ok: true } } });

    await userManagementService.getUsers({ search: 'dev' });
    await userManagementService.createUser({
      email: 'dev@example.com',
      username: 'dev',
      password: 'secret',
      role: 'developer',
    });
    await userManagementService.updateUser(1, { role: 'tester' });
    await userManagementService.deleteUser(1);
    await userManagementService.getUserWorkload(1);

    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/admin/users', {
      params: { search: 'dev' },
    });
    expect(mockedApi.post).toHaveBeenCalledWith('/api/admin/users', {
      email: 'dev@example.com',
      username: 'dev',
      password: 'secret',
      role: 'developer',
    });
    expect(mockedApi.put).toHaveBeenCalledWith('/api/admin/users/1', { role: 'tester' });
    expect(mockedApi.delete).toHaveBeenCalledWith('/api/admin/users/1');
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/admin/users/1/workload');
  });
});
