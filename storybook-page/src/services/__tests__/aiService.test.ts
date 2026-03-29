import api from '../api';
import { aiService } from '../aiService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('aiService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('应命中故事生成与 AI 配置接口', async () => {
    mockedApi.post
      .mockResolvedValueOnce({ data: { data: { title: 'AI Story' } } })
      .mockResolvedValueOnce({ data: { data: { provider: 'openai', model: 'gpt' } } });
    mockedApi.get.mockResolvedValueOnce({ data: { data: { config: { enabled: true } } } });
    mockedApi.put.mockResolvedValueOnce({ data: { data: { config: { enabled: false } } } });

    await aiService.generateStory({ requirement: '登录功能' });
    await aiService.getConfig();
    await aiService.updateConfig({ enabled: false });
    await aiService.testConfig({ model: 'gpt-4.1' });

    expect(mockedApi.post).toHaveBeenNthCalledWith(1, '/api/ai/generate-story', {
      requirement: '登录功能',
    });
    expect(mockedApi.get).toHaveBeenCalledWith('/api/admin/ai/config');
    expect(mockedApi.put).toHaveBeenCalledWith('/api/admin/ai/config', { enabled: false });
    expect(mockedApi.post).toHaveBeenNthCalledWith(2, '/api/admin/ai/config/test', {
      model: 'gpt-4.1',
    });
  });

  it('应命中拆分与 INVEST 检查接口', async () => {
    mockedApi.post.mockResolvedValueOnce({ data: { data: { story_id: 9, sub_stories: [] } } });
    mockedApi.get.mockResolvedValueOnce({ data: { data: { story_id: 9, invest_score: 0.8 } } });

    await aiService.splitStory(9, { target_count: 4 });
    await aiService.checkInvest(9);

    expect(mockedApi.post).toHaveBeenCalledWith('/api/ai/stories/9/split', {
      target_count: 4,
    });
    expect(mockedApi.get).toHaveBeenCalledWith('/api/ai/stories/9/invest-check');
  });
});
