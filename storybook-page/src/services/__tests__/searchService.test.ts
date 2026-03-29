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
