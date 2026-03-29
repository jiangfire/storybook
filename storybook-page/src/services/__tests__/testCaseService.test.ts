import api from '../api';
import { testCaseService } from '../testCaseService';

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api, { deep: true });

describe('testCaseService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('应命中测试用例列表、创建和状态更新接口', async () => {
    mockedApi.get.mockResolvedValueOnce({ data: { data: { test_cases: [] } } });
    mockedApi.post.mockResolvedValueOnce({ data: { data: { id: 12 } } });
    mockedApi.patch.mockResolvedValueOnce({ data: { data: { id: 12, status: 'passed' } } });

    await testCaseService.getStoryTestCases(5);
    await testCaseService.createTestCase(5, {
      title: 'case',
      steps: ['step-1'],
    });
    await testCaseService.updateTestCaseStatus(12, { status: 'passed' });

    expect(mockedApi.get).toHaveBeenCalledWith('/api/stories/5/test-cases');
    expect(mockedApi.post).toHaveBeenCalledWith('/api/stories/5/test-cases', {
      title: 'case',
      steps: ['step-1'],
    });
    expect(mockedApi.patch).toHaveBeenCalledWith('/api/test-cases/12/status', {
      status: 'passed',
    });
  });
});
