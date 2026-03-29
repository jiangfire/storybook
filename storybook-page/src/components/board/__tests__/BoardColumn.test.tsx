import { render, screen } from '@testing-library/react';
import type { StoryBoardItem } from '../../../types/models';
import { InboxIcon } from '../../ui/AppIcon';
import BoardColumn from '../BoardColumn';

vi.mock('@dnd-kit/core', () => ({
  useDroppable: () => ({
    setNodeRef: vi.fn(),
  }),
}));

vi.mock('@dnd-kit/sortable', () => ({
  SortableContext: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  verticalListSortingStrategy: {},
}));

vi.mock('../StoryCard', () => ({
  default: ({ story }: { story: StoryBoardItem }) => <div>StoryCard:{story.title}</div>,
}));

function createStory(id: number, title: string): StoryBoardItem {
  return {
    id,
    title,
    story_type: 'feature',
    status: 'backlog',
    priority: 2,
    position: id * 1024,
  };
}

describe('BoardColumn', () => {
  it('backlog 空列会显示创建故事提示', () => {
    render(<BoardColumn id="backlog" stories={[]} title="待办" icon={InboxIcon} count={0} />);

    expect(screen.getByText('暂无故事')).toBeInTheDocument();
    expect(screen.getByText('点击“创建故事”')).toBeInTheDocument();
    expect(screen.getByText('0')).toBeInTheDocument();
  });

  it('有故事时会渲染 StoryCard 列表', () => {
    render(
      <BoardColumn
        id="ready"
        stories={[createStory(1, '登录故事'), createStory(2, '支付故事')]}
        title="就绪"
        icon={InboxIcon}
        count={2}
      />
    );

    expect(screen.getByText('就绪')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('StoryCard:登录故事')).toBeInTheDocument();
    expect(screen.getByText('StoryCard:支付故事')).toBeInTheDocument();
  });
});
