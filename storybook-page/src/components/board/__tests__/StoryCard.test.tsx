import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { StoryBoardItem } from '../../../types/models';
import StoryCard from '../StoryCard';

const navigate = vi.fn();

vi.mock('@dnd-kit/sortable', () => ({
  useSortable: ({ id }: { id: number }) => ({
    attributes: { 'data-sortable-id': id },
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    transition: 'transform 200ms ease',
    isDragging: false,
  }),
}));

vi.mock('@dnd-kit/utilities', () => ({
  CSS: {
    Transform: {
      toString: () => 'translate3d(0, 0, 0)',
    },
  },
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => navigate,
  };
});

function renderCard(story: StoryBoardItem) {
  return render(<StoryCard story={story} />);
}

describe('StoryCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('展示验收标准摘要、点数、负责人并支持进入详情', async () => {
    const user = userEvent.setup();

    renderCard({
      id: 18,
      title: '登录故事',
      story_type: 'feature',
      status: 'ready',
      priority: 3,
      story_points: 5,
      acceptance_criteria_summary: {
        total: 4,
        passed: 2,
        pending: 2,
        completion_percentage: 50,
      },
      assigned_to: {
        id: 11,
        email: 'dev@example.com',
        role: 'developer',
        created_at: '2026-03-29T00:00:00Z',
      },
    });

    expect(screen.getByText('功能')).toBeInTheDocument();
    expect(screen.getByText('高')).toBeInTheDocument();
    expect(screen.getByText('2/4')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByTitle('dev@example.com')).toHaveTextContent('D');

    await user.click(screen.getByRole('button', { name: '详情' }));

    expect(navigate).toHaveBeenCalledWith('/stories/18');
  });

  it('无负责人时显示未分配，并从验收标准明细计算进度', () => {
    renderCard({
      id: 19,
      title: '支付故事',
      story_type: 'bug',
      status: 'backlog',
      priority: 1,
      acceptance_criteria: [
        { id: 'ac-1', description: 'A', status: 'passed', order: 1 },
        { id: 'ac-2', description: 'B', status: 'pending', order: 2 },
      ],
    });

    expect(screen.getByText('Bug')).toBeInTheDocument();
    expect(screen.getByText('低')).toBeInTheDocument();
    expect(screen.getByText('1/2')).toBeInTheDocument();
    expect(screen.getByTitle('未分配')).toHaveTextContent('?');
  });
});
