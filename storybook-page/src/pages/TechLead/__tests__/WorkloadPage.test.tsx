import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { UserWorkload } from '../../../types/models';
import { techLeadService } from '../../../services/techLeadService';
import WorkloadPage from '../WorkloadPage';

const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showError,
  }),
}));

vi.mock('../../../services/techLeadService', () => ({
  techLeadService: {
    getWorkload: vi.fn(),
    getMyProjects: vi.fn(),
  },
}));

const mockedTechLeadService = vi.mocked(techLeadService, { deep: true });

const NOW = '2026-03-29T00:00:00Z';

function createWorkload(
  id: number,
  overrides: Partial<UserWorkload> = {}
): UserWorkload {
  return {
    user: {
      id,
      email: `user${id}@example.com`,
      role: 'developer',
      created_at: NOW,
    },
    active_stories: 1,
    active_tasks: 2,
    total_story_points: 5,
    estimated_hours: 8,
    completion_rate: 75,
    ...overrides,
  };
}

describe('WorkloadPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockedTechLeadService.getWorkload.mockResolvedValue({
      workloads: [
        createWorkload(1, {
          active_stories: 2,
          total_story_points: 16,
          estimated_hours: 20,
          completion_rate: 90,
        }),
        createWorkload(2, {
          active_stories: 1,
          active_tasks: 1,
          total_story_points: 3,
          estimated_hours: 4,
          completion_rate: 40,
        }),
      ],
    });
    mockedTechLeadService.getMyProjects.mockResolvedValue({
      projects: [
        { id: 1, name: 'Alpha', agile_mode: 'kanban', pending_stories: 1 },
        { id: 2, name: 'Beta', agile_mode: 'scrum', pending_stories: 0 },
      ],
    });
  });

  it('会展示团队汇总指标和成员负载等级', async () => {
    render(<WorkloadPage />);

    await waitFor(() => {
      expect(mockedTechLeadService.getWorkload).toHaveBeenCalledWith(undefined);
      expect(mockedTechLeadService.getMyProjects).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByText('团队成员')).toBeInTheDocument();
    expect(screen.getByText('团队成员').parentElement).toHaveTextContent('2');
    expect(screen.getAllByText('高负载').length).toBeGreaterThan(0);
    expect(screen.getAllByText('低负载').length).toBeGreaterThan(0);
    expect(screen.getByText('24.0')).toBeInTheDocument();
  });

  it('切换项目后会按项目重新拉取工作负载', async () => {
    const user = userEvent.setup();

    render(<WorkloadPage />);

    await screen.findByText('工作负载');
    await user.selectOptions(screen.getByRole('combobox'), '2');

    await waitFor(() => {
      expect(mockedTechLeadService.getWorkload).toHaveBeenLastCalledWith(2);
    });
  });

  it('加载失败时会通过 toast 提示错误', async () => {
    mockedTechLeadService.getWorkload.mockRejectedValueOnce(new Error('接口不可用'));

    render(<WorkloadPage />);

    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('接口不可用');
    });
  });
});
