import api from '../api';
import { storyService } from '../storyService';

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

describe('storyService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('board / assign / sprint / activities / claim / release 接口路径正确', async () => {
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { project_id: 1, columns: [] } } })
      .mockResolvedValueOnce({ data: { data: { activities: [] } } });
    mockedApi.patch
      .mockResolvedValueOnce({ data: { data: { id: 3 } } })
      .mockResolvedValueOnce({ data: { data: { id: 3 } } });
    mockedApi.post.mockResolvedValueOnce({ data: { data: { id: 3 } } });
    mockedApi.delete.mockResolvedValueOnce({ data: { data: { id: 3 } } });

    await storyService.getBoardData(1);
    await storyService.assignStory(3, { assigned_to: 9 });
    await storyService.planToSprint(3, { sprint_id: 6 });
    await storyService.getActivities(3, { page: 2 });
    await storyService.claimStory(3);
    await storyService.releaseStory(3);

    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/projects/1/board');
    expect(mockedApi.patch).toHaveBeenNthCalledWith(1, '/api/stories/3/assignee', {
      assigned_to: 9,
    });
    expect(mockedApi.patch).toHaveBeenNthCalledWith(2, '/api/stories/3/sprint', {
      sprint_id: 6,
    });
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/stories/3/activities', {
      params: { page: 2 },
    });
    expect(mockedApi.post).toHaveBeenCalledWith('/api/stories/3/claim');
    expect(mockedApi.delete).toHaveBeenCalledWith('/api/stories/3/claim');
  });

  it('create/update/status/ac/code-ref 等故事接口应返回 data.data', async () => {
    mockedApi.post
      .mockResolvedValueOnce({ data: { data: { id: 10 } } })
      .mockResolvedValueOnce({ data: { data: { ok: true } } });
    mockedApi.put.mockResolvedValueOnce({ data: { data: { id: 10 } } });
    mockedApi.patch
      .mockResolvedValueOnce({ data: { data: { id: 10, status: 'ready' } } })
      .mockResolvedValueOnce({ data: { data: { ok: true } } });

    const created = await storyService.createStory(2, {
      title: '登录',
      story_type: 'feature',
    });
    const updated = await storyService.updateStory(10, { title: '登录 v2' });
    const status = await storyService.updateStoryStatus(10, { status: 'ready', position: 5 });
    await storyService.updateACStatus(10, 'ac-1', { status: 'passed', evidence: 'e2e' });
    await storyService.addCodeRef(10, { reference: 'src/story.ts:12' });

    expect(created.id).toBe(10);
    expect(updated.id).toBe(10);
    expect(status.status).toBe('ready');
    expect(mockedApi.post).toHaveBeenNthCalledWith(1, '/api/projects/2/stories', {
      title: '登录',
      story_type: 'feature',
    });
    expect(mockedApi.put).toHaveBeenCalledWith('/api/stories/10', { title: '登录 v2' });
    expect(mockedApi.patch).toHaveBeenNthCalledWith(1, '/api/stories/10/status', {
      status: 'ready',
      position: 5,
    });
    expect(mockedApi.patch).toHaveBeenNthCalledWith(
      2,
      '/api/stories/10/acceptance-criteria/ac-1',
      { status: 'passed', evidence: 'e2e' }
    );
    expect(mockedApi.post).toHaveBeenNthCalledWith(2, '/api/stories/10/code-refs', {
      reference: 'src/story.ts:12',
    });
  });
});
