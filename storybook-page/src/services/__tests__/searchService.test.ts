import apiClient from '../api';
import { searchService } from '../searchService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
  },
}));

const mockedApiClient = vi.mocked(apiClient, { deep: true });

describe('searchService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('search 会命中关键字搜索接口', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: {
          projects: [],
          stories: [],
          bugs: [],
        },
      },
    });

    await searchService.search({ q: 'login', type: 'all', limit: 8 });

    expect(mockedApiClient.get).toHaveBeenCalledWith('/api/search', {
      params: { q: 'login', type: 'all', limit: 8 },
    });
  });

  it('searchSemanticStories 会传递默认 limit', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: {
          stories: [],
        },
      },
    });

    await searchService.searchSemanticStories('payment');

    expect(mockedApiClient.get).toHaveBeenCalledWith('/api/search/semantic', {
      params: { q: 'payment', limit: 5 },
    });
  });

  it('search 会携带 assignee 负责人过滤参数', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: {
          projects: [],
          stories: [],
          bugs: [],
        },
      },
    });

    await searchService.search({ q: 'login', assignee: 5 });

    expect(mockedApiClient.get).toHaveBeenCalledWith('/api/search', {
      params: { q: 'login', assignee: 5 },
    });
  });

  it('listAssignees 会命中负责人候选接口并返回用户列表', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: {
          users: [{ id: 2, email: 'dev@test.dev' }],
        },
      },
    });

    const users = await searchService.listAssignees();

    expect(mockedApiClient.get).toHaveBeenCalledWith('/api/search/assignees');
    expect(users).toEqual([{ id: 2, email: 'dev@test.dev' }]);
  });

  it('getCapabilities 会命中能力接口', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: {
        data: {
          semantic_enabled: true,
        },
      },
    });

    await searchService.getCapabilities();

    expect(mockedApiClient.get).toHaveBeenCalledWith('/api/search/capabilities');
  });
});
