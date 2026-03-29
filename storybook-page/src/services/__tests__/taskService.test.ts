import api from '../api';
import { taskService } from '../taskService';

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

describe('taskService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('应命中任务列表、创建、拆分、详情、更新、删除、状态、进度、领取、释放和代码引用接口', async () => {
    mockedApi.get
      .mockResolvedValueOnce({ data: { data: { tasks: [] } } })
      .mockResolvedValueOnce({ data: { data: { id: 8 } } });
    mockedApi.post
      .mockResolvedValueOnce({ data: { data: { id: 8 } } })
      .mockResolvedValueOnce({ data: { data: { created_count: 2, tasks: [] } } })
      .mockResolvedValueOnce({ data: { data: { id: 8 } } })
      .mockResolvedValueOnce({ data: { data: { task_id: 8, code_references: ['a.ts:1'] } } });
    mockedApi.put.mockResolvedValueOnce({ data: { data: { id: 8 } } });
    mockedApi.patch
      .mockResolvedValueOnce({ data: { data: { id: 8, status: 'done' } } })
      .mockResolvedValueOnce({ data: { data: { id: 8, progress: 100 } } });
    mockedApi.delete
      .mockResolvedValueOnce({ data: { data: { ok: true } } })
      .mockResolvedValueOnce({ data: { data: { id: 8 } } });

    await taskService.getStoryTasks(2, { status: 'todo' });
    await taskService.createTask(2, { title: '任务' });
    await taskService.splitFromAC(2);
    await taskService.getTask(8);
    await taskService.updateTask(8, { title: '任务 v2' });
    await taskService.deleteTask(8);
    await taskService.updateTaskStatus(8, { status: 'done' });
    await taskService.updateTaskProgress(8, { progress: 100 });
    await taskService.claimTask(8);
    await taskService.releaseTask(8);
    await taskService.addCodeRef(8, 'src/task.ts:1');

    expect(mockedApi.get).toHaveBeenNthCalledWith(1, '/api/stories/2/tasks', {
      params: { status: 'todo' },
    });
    expect(mockedApi.post).toHaveBeenNthCalledWith(1, '/api/stories/2/tasks', { title: '任务' });
    expect(mockedApi.post).toHaveBeenNthCalledWith(2, '/api/stories/2/tasks/split-from-ac');
    expect(mockedApi.get).toHaveBeenNthCalledWith(2, '/api/tasks/8');
    expect(mockedApi.put).toHaveBeenCalledWith('/api/tasks/8', { title: '任务 v2' });
    expect(mockedApi.delete).toHaveBeenNthCalledWith(1, '/api/tasks/8');
    expect(mockedApi.patch).toHaveBeenNthCalledWith(1, '/api/tasks/8/status', { status: 'done' });
    expect(mockedApi.patch).toHaveBeenNthCalledWith(2, '/api/tasks/8/progress', {
      progress: 100,
    });
    expect(mockedApi.post).toHaveBeenNthCalledWith(3, '/api/tasks/8/claim');
    expect(mockedApi.delete).toHaveBeenNthCalledWith(2, '/api/tasks/8/claim');
    expect(mockedApi.post).toHaveBeenNthCalledWith(4, '/api/tasks/8/code-refs', {
      reference: 'src/task.ts:1',
    });
  });
});
