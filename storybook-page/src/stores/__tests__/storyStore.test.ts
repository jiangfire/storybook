import { storyService } from '../../services/storyService';
import { useStoryStore } from '../storyStore';

vi.mock('../../services/storyService', () => ({
  storyService: {
    getBoardData: vi.fn(),
    getStory: vi.fn(),
    getActivities: vi.fn(),
    updateStoryStatus: vi.fn(),
    claimStory: vi.fn(),
    releaseStory: vi.fn(),
    updateACStatus: vi.fn(),
  },
}));

const mockedStoryService = vi.mocked(storyService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function resetStore() {
  useStoryStore.setState({
    boardData: {
      pending: [],
      backlog: [],
      ready: [],
      in_progress: [],
      test: [],
      done: [],
    },
    currentStory: null,
    activities: [],
    isLoading: false,
    isUpdating: false,
    error: null,
  });
}

describe('storyStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it('fetchBoardData 会补齐固定列，并把 assignee 归一化到 assigned_to', async () => {
    mockedStoryService.getBoardData.mockResolvedValue({
      project_id: 1,
      columns: [
        {
          position: 0,
          name: '待审批',
          status: 'pending',
          count: 1,
          stories: [
            {
              id: 1,
              title: 'Pending Story',
              story_type: 'feature',
              status: 'pending',
              priority: 2,
              assignee: {
                id: 8,
                email: 'dev@example.com',
                role: 'developer',
                created_at: NOW,
              },
            },
          ],
        },
      ],
    });

    await useStoryStore.getState().fetchBoardData(1);

    const { boardData } = useStoryStore.getState();
    expect(boardData.pending).toHaveLength(1);
    expect(boardData.pending[0].assigned_to?.id).toBe(8);
    expect(boardData.backlog).toEqual([]);
    expect(boardData.done).toEqual([]);
  });

  it('updateStoryStatus 成功时会乐观更新状态', async () => {
    useStoryStore.setState({
      boardData: {
        pending: [],
        backlog: [
          {
            id: 3,
            title: 'Backlog Story',
            story_type: 'feature',
            status: 'backlog',
            priority: 2,
            position: 100,
          },
        ],
        ready: [],
        in_progress: [],
        test: [],
        done: [],
      },
    });
    mockedStoryService.updateStoryStatus.mockResolvedValue({});

    await useStoryStore.getState().updateStoryStatus(3, { status: 'ready', position: 0 });

    const { boardData, isUpdating } = useStoryStore.getState();
    expect(boardData.backlog).toEqual([]);
    expect(boardData.ready[0]).toMatchObject({ id: 3, status: 'ready', position: 0 });
    expect(isUpdating).toBe(false);
  });

  it('updateStoryStatus 失败时会回滚并保留错误信息', async () => {
    useStoryStore.setState({
      boardData: {
        pending: [],
        backlog: [
          {
            id: 3,
            title: 'Backlog Story',
            story_type: 'feature',
            status: 'backlog',
            priority: 2,
            position: 100,
          },
        ],
        ready: [],
        in_progress: [],
        test: [],
        done: [],
      },
    });
    mockedStoryService.updateStoryStatus.mockRejectedValue(new Error('状态更新失败'));

    await expect(
      useStoryStore.getState().updateStoryStatus(3, { status: 'ready', position: 0 })
    ).rejects.toBeTruthy();

    const { boardData, error } = useStoryStore.getState();
    expect(boardData.backlog[0]).toMatchObject({ id: 3, status: 'backlog' });
    expect(boardData.ready).toEqual([]);
    expect(error).toBe('状态更新失败');
  });

  it('claimStory 会同步更新 currentStory 和看板状态', async () => {
    useStoryStore.setState({
      boardData: {
        pending: [],
        backlog: [],
        ready: [
          {
            id: 5,
            title: 'Ready Story',
            story_type: 'feature',
            status: 'ready',
            priority: 1,
            position: 200,
          },
        ],
        in_progress: [],
        test: [],
        done: [],
      },
      currentStory: {
        id: 5,
        project_id: 1,
        title: 'Ready Story',
        story_type: 'feature',
        status: 'ready',
        priority: 1,
        position: 200,
        created_by: {
          id: 1,
          email: 'pm@example.com',
          role: 'product',
          created_at: NOW,
        },
        acceptance_criteria: [],
        created_at: NOW,
        updated_at: NOW,
      },
    });
    mockedStoryService.claimStory.mockResolvedValue({
      assigned_to: {
        id: 9,
        email: 'dev@example.com',
      },
      status: 'in_progress',
    });

    await useStoryStore.getState().claimStory(5);

    const { boardData, currentStory } = useStoryStore.getState();
    expect(boardData.ready).toEqual([]);
    expect(boardData.in_progress[0]).toMatchObject({
      id: 5,
      status: 'in_progress',
      assigned_to: { id: 9, email: 'dev@example.com' },
    });
    expect(currentStory?.status).toBe('in_progress');
    expect(currentStory?.assigned_to?.id).toBe(9);
  });

  it('releaseStory 和 updateACStatus 会分别同步负责人状态与 AC 状态', async () => {
    useStoryStore.setState({
      boardData: {
        pending: [],
        backlog: [],
        ready: [],
        in_progress: [
          {
            id: 6,
            title: 'Working Story',
            story_type: 'feature',
            status: 'in_progress',
            priority: 1,
            position: 300,
            assigned_to: {
              id: 9,
              email: 'dev@example.com',
              role: 'developer',
              created_at: NOW,
            },
          },
        ],
        test: [],
        done: [],
      },
      currentStory: {
        id: 6,
        project_id: 1,
        title: 'Working Story',
        story_type: 'feature',
        status: 'in_progress',
        priority: 1,
        position: 300,
        assigned_to: {
          id: 9,
          email: 'dev@example.com',
          role: 'developer',
          created_at: NOW,
        },
        created_by: {
          id: 1,
          email: 'pm@example.com',
          role: 'product',
          created_at: NOW,
        },
        acceptance_criteria: [
          {
            id: 'ac-1',
            description: 'Given 用户存在',
            status: 'pending',
            order: 1,
          },
        ],
        created_at: NOW,
        updated_at: NOW,
      },
    });
    mockedStoryService.releaseStory.mockResolvedValue({ status: 'ready' });
    mockedStoryService.updateACStatus.mockResolvedValue(undefined);

    await useStoryStore.getState().releaseStory(6);
    await useStoryStore.getState().updateACStatus(6, 'ac-1', 'passed', 'e2e');

    const { boardData, currentStory } = useStoryStore.getState();
    expect(boardData.in_progress).toEqual([]);
    expect(boardData.ready[0]).toMatchObject({ id: 6, status: 'ready', assigned_to: undefined });
    expect(currentStory?.assigned_to).toBeUndefined();
    expect(currentStory?.status).toBe('ready');
    expect(currentStory?.acceptance_criteria[0]).toMatchObject({
      id: 'ac-1',
      status: 'passed',
      evidence: 'e2e',
    });
    expect(mockedStoryService.updateACStatus).toHaveBeenCalledWith(6, 'ac-1', {
      status: 'passed',
      evidence: 'e2e',
    });
  });

  it('fetchStory 成功和失败时都会正确维护 loading 与错误状态', async () => {
    mockedStoryService.getStory.mockResolvedValueOnce({
      id: 8,
      project_id: 1,
      title: 'Story Detail',
      story_type: 'feature',
      status: 'ready',
      priority: 1,
      position: 400,
      created_by: {
        id: 1,
        email: 'pm@example.com',
        role: 'product',
        created_at: NOW,
      },
      acceptance_criteria: [],
      created_at: NOW,
      updated_at: NOW,
    });

    await useStoryStore.getState().fetchStory(8);

    expect(useStoryStore.getState().currentStory?.id).toBe(8);
    expect(useStoryStore.getState().isLoading).toBe(false);

    mockedStoryService.getStory.mockRejectedValueOnce(new Error('故事加载失败'));
    await useStoryStore.getState().fetchStory(9);

    expect(useStoryStore.getState().error).toBe('故事加载失败');
    expect(useStoryStore.getState().isLoading).toBe(false);
  });

  it('fetchActivities 失败时会记录错误，clearCurrentStory 和 clearError 可清理状态', async () => {
    useStoryStore.setState({
      currentStory: {
        id: 10,
        project_id: 1,
        title: 'Story Activities',
        story_type: 'feature',
        status: 'test',
        priority: 1,
        position: 500,
        created_by: {
          id: 1,
          email: 'pm@example.com',
          role: 'product',
          created_at: NOW,
        },
        acceptance_criteria: [],
        created_at: NOW,
        updated_at: NOW,
      },
      activities: [{ id: 1, entity_type: 'story', action: 'created', created_at: NOW } as never],
    });
    mockedStoryService.getActivities.mockRejectedValueOnce(new Error('活动加载失败'));

    await useStoryStore.getState().fetchActivities(10);

    expect(useStoryStore.getState().error).toBe('活动加载失败');

    useStoryStore.getState().clearCurrentStory();
    expect(useStoryStore.getState().currentStory).toBeNull();
    expect(useStoryStore.getState().activities).toEqual([]);

    useStoryStore.getState().clearError();
    expect(useStoryStore.getState().error).toBeNull();
  });

  it('claimStory 失败时会保留错误并继续抛出异常', async () => {
    const error = new Error('领取失败');
    mockedStoryService.claimStory.mockRejectedValueOnce(error);

    await expect(useStoryStore.getState().claimStory(99)).rejects.toBe(error);

    expect(useStoryStore.getState().error).toBe('领取失败');
    expect(useStoryStore.getState().isUpdating).toBe(false);
  });
});
