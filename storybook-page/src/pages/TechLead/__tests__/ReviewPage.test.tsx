import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Story } from '../../../types/models';
import { techLeadService } from '../../../services/techLeadService';
import ReviewPage from '../ReviewPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/techLeadService', () => ({
  techLeadService: {
    getPendingStories: vi.fn(),
    getMyProjects: vi.fn(),
    reviewStory: vi.fn(),
  },
}));

const mockedTechLeadService = vi.mocked(techLeadService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function createStory(id: number, overrides: Partial<Story> = {}): Story {
  return {
    id,
    project_id: 5,
    project: { id: 5, name: 'Alpha' },
    title: '登录故事',
    description: '实现登录',
    story_type: 'feature',
    status: 'pending',
    review_status: 'pending',
    priority: 2,
    story_points: 3,
    position: 1024,
    created_by: {
      id: 1,
      email: 'product@example.com',
      role: 'product',
      created_at: NOW,
    },
    acceptance_criteria: [],
    created_at: NOW,
    updated_at: NOW,
    ...overrides,
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/techlead/review']}>
      <Routes>
        <Route path="/techlead/review" element={<ReviewPage />} />
        <Route path="/stories/:id" element={<div>Story Detail Page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('ReviewPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockedTechLeadService.getPendingStories.mockResolvedValue({
      stories: [createStory(18)],
      total: 1,
      page: 1,
      limit: 20,
    });
    mockedTechLeadService.getMyProjects.mockResolvedValue({
      projects: [
        { id: 5, name: 'Alpha', agile_mode: 'kanban', pending_stories: 1 },
        { id: 9, name: 'Beta', agile_mode: 'scrum', pending_stories: 2 },
      ],
    });
    mockedTechLeadService.reviewStory.mockResolvedValue({});
  });

  it('拒绝故事时必须填写原因', async () => {
    const user = userEvent.setup();

    renderPage();

    expect(await screen.findByText('登录故事')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '拒绝' }));
    await user.click(screen.getByRole('button', { name: '确认拒绝' }));

    expect(await screen.findByText('拒绝审批必须填写原因')).toBeInTheDocument();
    expect(mockedTechLeadService.reviewStory).not.toHaveBeenCalled();
  });

  it('审批通过后会调用 review 接口并刷新列表', async () => {
    const user = userEvent.setup();

    renderPage();

    expect(await screen.findByText('登录故事')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '通过' }));
    await user.click(screen.getByRole('button', { name: '确认通过' }));

    await waitFor(() => {
      expect(mockedTechLeadService.reviewStory).toHaveBeenCalledWith(18, {
        approved: true,
        comment: '',
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('审批已通过');
    await waitFor(() => {
      expect(mockedTechLeadService.getPendingStories).toHaveBeenCalledTimes(2);
    });
  });

  it('切换项目筛选时会按项目重新拉取待审批故事', async () => {
    const user = userEvent.setup();

    renderPage();

    await screen.findByText('登录故事');
    await user.selectOptions(screen.getByRole('combobox'), '9');

    await waitFor(() => {
      expect(mockedTechLeadService.getPendingStories).toHaveBeenLastCalledWith({
        project_id: 9,
        search: undefined,
      });
    });
  });

  it('拒绝故事时会携带原因提交并刷新列表', async () => {
    const user = userEvent.setup();

    renderPage();

    expect(await screen.findByText('登录故事')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '拒绝' }));
    await user.type(screen.getByPlaceholderText('请输入拒绝原因...'), '验收标准不完整');
    await user.click(screen.getByRole('button', { name: '确认拒绝' }));

    await waitFor(() => {
      expect(mockedTechLeadService.reviewStory).toHaveBeenCalledWith(18, {
        approved: false,
        comment: '验收标准不完整',
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('已拒绝该故事');
  });

  it('点击搜索按钮会按关键字重新拉取待审批故事', async () => {
    const user = userEvent.setup();

    renderPage();

    await screen.findByText('登录故事');

    const input = screen.getByPlaceholderText('搜索故事标题...');
    fireEvent.change(input, { target: { value: '登录' } });

    await waitFor(() => {
      expect(mockedTechLeadService.getPendingStories).toHaveBeenLastCalledWith({
        project_id: undefined,
        search: '登录',
      });
    });

    const callCountBeforeSearch = mockedTechLeadService.getPendingStories.mock.calls.length;
    await user.click(screen.getByRole('button', { name: '搜索' }));

    await waitFor(() => {
      expect(mockedTechLeadService.getPendingStories.mock.calls.length).toBeGreaterThan(
        callCountBeforeSearch
      );
    });
  });

  it('加载失败时会提示错误并回退到空状态', async () => {
    mockedTechLeadService.getPendingStories.mockRejectedValueOnce(new Error('审批列表加载失败'));

    renderPage();

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('审批列表加载失败');
    });

    expect(screen.getByText('没有待审批的故事')).toBeInTheDocument();
  });
});
