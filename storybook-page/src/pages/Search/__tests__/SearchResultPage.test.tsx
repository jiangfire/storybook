import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { searchService } from '../../../services/searchService';
import SearchResultPage from '../SearchResultPage';

const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showError,
    showSuccess: vi.fn(),
  }),
}));

vi.mock('../../../services/searchService', () => ({
  searchService: {
    search: vi.fn(),
    searchSemanticStories: vi.fn(),
    getCapabilities: vi.fn(),
    listAssignees: vi.fn(),
  },
}));

const mockedSearchService = vi.mocked(searchService, { deep: true });

function renderPage(initialEntry = '/search?q=登录') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/search" element={<SearchResultPage />} />
        <Route path="/stories/:id" element={<div>Story Page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('SearchResultPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedSearchService.listAssignees.mockResolvedValue([
      { id: 2, email: 'dev@test.dev' },
      { id: 3, email: 'qa@test.dev' },
    ]);
    mockedSearchService.search.mockResolvedValue({
      projects: [{ id: 3, name: '登录重构', description: '', created_at: '2026-01-01T00:00:00Z' }],
      stories: [
        {
          id: 12,
          project_id: 3,
          title: '支持手机号登录',
          story_type: 'feature',
          status: 'done',
          priority: 2,
          updated_at: '2026-01-02T00:00:00Z',
        },
      ],
      bugs: [],
    });
  });

  it('按 URL 关键词搜索并分块展示结果与计数', async () => {
    renderPage();

    expect(await screen.findByText(/「登录」共 2 条结果（项目 1 · 故事 1 · 缺陷 0）/)).toBeInTheDocument();
    expect(screen.getByText('登录重构')).toBeInTheDocument();
    expect(screen.getByText('#12 支持手机号登录')).toBeInTheDocument();
    expect(mockedSearchService.search).toHaveBeenCalledWith(
      expect.objectContaining({ q: '登录', type: 'all', limit: 50 })
    );
  });

  it('切换类型与状态过滤会更新查询参数并重新搜索', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('#12 支持手机号登录');

    await user.click(screen.getByRole('button', { name: '故事' }));
    await waitFor(() => {
      expect(mockedSearchService.search).toHaveBeenLastCalledWith(
        expect.objectContaining({ type: 'story' })
      );
    });

    await user.click(screen.getByRole('button', { name: '进行中' }));
    await waitFor(() => {
      expect(mockedSearchService.search).toHaveBeenLastCalledWith(
        expect.objectContaining({ type: 'story', status: ['in_progress'] })
      );
    });
  });

  it('URL 带 assignee 时按负责人过滤并在下拉中选中', async () => {
    renderPage('/search?q=登录&assignee=2');

    expect(await screen.findByText('#12 支持手机号登录')).toBeInTheDocument();
    expect(mockedSearchService.search).toHaveBeenCalledWith(
      expect.objectContaining({ q: '登录', assignee: 2 })
    );
    await waitFor(() => {
      expect(screen.getByLabelText('负责人')).toHaveValue('2');
    });
  });

  it('切换负责人下拉会更新过滤并重新搜索，清除后不再携带', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('#12 支持手机号登录');

    await user.selectOptions(screen.getByLabelText('负责人'), '3');
    await waitFor(() => {
      expect(mockedSearchService.search).toHaveBeenLastCalledWith(
        expect.objectContaining({ q: '登录', assignee: 3 })
      );
    });

    await user.selectOptions(screen.getByLabelText('负责人'), '');
    await waitFor(() => {
      const lastCall = mockedSearchService.search.mock.lastCall?.[0];
      expect(lastCall).toMatchObject({ q: '登录' });
      expect(lastCall?.assignee).toBeUndefined();
    });
  });

  it('空关键词时提示输入且不发起请求', () => {
    renderPage('/search');

    expect(screen.getByText('请输入关键词开始搜索')).toBeInTheDocument();
    expect(mockedSearchService.search).not.toHaveBeenCalled();
  });
});
