import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { Story } from '../../../../types/models';
import { storyService } from '../../../../services/storyService';
import { useAuthStore } from '../../../../stores/authStore';
import ProjectStoriesSection from '../ProjectStoriesSection';

const NOW = '2026-03-29T00:00:00Z';

vi.mock('../../../../services/storyService', () => ({
  storyService: {
    getStories: vi.fn(),
  },
}));

const mockedStoryService = vi.mocked(storyService, { deep: true });

function makeStory(overrides: Partial<Story> = {}): Story {
  return {
    id: 1,
    project_id: 1,
    title: '支持手机号登录',
    story_type: 'feature',
    status: 'pending',
    review_status: 'pending',
    priority: 3,
    position: 1,
    created_by: { id: 1, email: 'pm@example.com', role: 'product', created_at: NOW },
    acceptance_criteria: [],
    created_at: NOW,
    updated_at: NOW,
    ...overrides,
  };
}

function setAuthUser(role: 'product' | 'developer') {
  useAuthStore.setState({
    user: { id: 7, email: `${role}@example.com`, role, created_at: NOW },
    token: 'token',
    isAuthenticated: true,
    isLoading: false,
    error: null,
  });
}

function renderSection(props: Partial<{ projectId: number; canCreateStory: boolean }> = {}) {
  return render(
    <MemoryRouter>
      <ProjectStoriesSection projectId={props.projectId ?? 1} canCreateStory={props.canCreateStory ?? true} />
    </MemoryRouter>
  );
}

describe('ProjectStoriesSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAuthUser('product');
  });

  it('渲染需求列表并显示审批状态徽章', async () => {
    mockedStoryService.getStories.mockResolvedValue({
      stories: [
        makeStory(),
        makeStory({
          id: 2,
          title: '导出报表功能',
          status: 'in_progress',
          priority: 1,
          story_points: 5,
          assigned_to: { id: 9, email: 'dev@example.com', role: 'developer', created_at: NOW },
        }),
      ],
      total: 2,
    });

    renderSection();

    expect(await screen.findByText('支持手机号登录')).toBeInTheDocument();
    expect(screen.getByText('导出报表功能')).toBeInTheDocument();
    expect(screen.getAllByText('待审批').length).toBeGreaterThan(0);
    expect(screen.getByText('dev@example.com')).toBeInTheDocument();
    expect(screen.getByText('5 点')).toBeInTheDocument();
  });

  it('切换状态筛选会带参数重新请求', async () => {
    const user = userEvent.setup();
    mockedStoryService.getStories.mockResolvedValue({ stories: [], total: 0 });

    renderSection();

    await waitFor(() => {
      expect(mockedStoryService.getStories).toHaveBeenCalledWith(1, {
        status: undefined,
        sort_by: 'priority',
        order: 'desc',
      });
    });

    await user.selectOptions(screen.getByLabelText('状态'), 'in_progress');

    await waitFor(() => {
      expect(mockedStoryService.getStories).toHaveBeenCalledWith(1, {
        status: 'in_progress',
        sort_by: 'priority',
        order: 'desc',
      });
    });
  });

  it('PM 看到空态创建引导，其他角色不显示', async () => {
    mockedStoryService.getStories.mockResolvedValue({ stories: [], total: 0 });

    const first = renderSection();
    expect(
      await screen.findByRole('link', { name: '创建第一个故事' })
    ).toHaveAttribute('href', '/projects/1/stories/new');
    first.unmount();

    setAuthUser('developer');
    mockedStoryService.getStories.mockClear();
    mockedStoryService.getStories.mockResolvedValue({ stories: [], total: 0 });

    renderSection({ canCreateStory: false });

    expect(await screen.findByText('还没有故事')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '创建第一个故事' })).not.toBeInTheDocument();
  });
});
